package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var vmInternalIP string

func init() {
	vmInternalIP = os.Getenv("VM_INTERNAL_IP")
	functions.HTTP("BackgroundWorker", backgroundWorker)
}

// PubSubPushRequest maps the GCP Pub/Sub push wrapper schema
type PubSubPushRequest struct {
	Message struct {
		// Go's json.Unmarshal automatically decodes Base64 strings into []byte
		Data []byte `json:"data"`
	} `json:"message"`
}

// MessagePayload structures the data sent from your fast-ingress service
type MessagePayload struct {
	Body     json.RawMessage `json:"body"`
	OriginIP string          `json:"originIp"`
	Path     string          `json:"path"`
}

func backgroundWorker(w http.ResponseWriter, r *http.Request) {
	log.Println("=========== Starting Background Worker ===========")
	defer log.Println("=========== End Background Worker ===========")

	// 1. Verify the request has the native wrapper structure that GCP Pub/Sub pushes use
	var pushReq PubSubPushRequest
	if err := json.NewDecoder(r.Body).Decode(&pushReq); err != nil || len(pushReq.Message.Data) == 0 {
		log.Printf("Validation Failed: Incoming request does not match the GCP Pub/Sub push scheme. Err: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Invalid Pub/Sub structure."))
		return
	}

	// 2. Extract and parse the inner payload
	var payload MessagePayload
	if err := json.Unmarshal(pushReq.Message.Data, &payload); err != nil {
		log.Printf("Execution Failed: Could not parse JSON payload: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Worker execution failed."))
		return
	}

	// 3. Schema Validation
	if len(payload.Body) == 0 || string(payload.Body) == "null" {
		log.Println("Validation Failed: Decoded payload is missing the core \"body\" property.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Missing required body content."))
		return
	}

	originIP := payload.OriginIP
	if originIP == "" {
		originIP = "Unknown"
	}
	log.Printf("Validation Passed: Secure packet received from IP %s", originIP)

	// 4. Trigger the Java Service inside the VM
	targetURL := fmt.Sprintf("http://%s:9090/saveNotification", vmInternalIP)
	log.Printf("Sending payload to VM at: %s", targetURL)

	// Send the inner body directly via POST
	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(payload.Body))
	if err != nil {
		log.Printf("Execution Failed: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Worker execution failed."))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Execution Failed: Java service responded with status: %d", resp.StatusCode)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Worker execution failed."))
		return
	}

	responseData, _ := io.ReadAll(resp.Body)
	log.Printf("Successfully triggered VM. Response: %s", string(responseData))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Worker executed successfully."))
}
