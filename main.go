package main

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"k8s.io/api/admission/v1beta1"
	k8v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// ErrNameNotProvided is thrown when a name is not provided
	ErrNameNotProvided = errors.New("no name was provided in the HTTP body")
)

// Handler is your Lambda function handler
// It uses Amazon API Gateway request/responses provided by the aws-lambda-go/events package,
// However you could use other event sources (S3, Kinesis etc), or JSON-decoded primitive types such as 'string'.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Processing Lambda request %s\n", request.RequestContext.RequestID)

	// If no name is provided in the HTTP request body, throw an error

	body := request.Body
	log.Printf("Processing Lambda request %s\n", body)

	admissionrequest := v1beta1.AdmissionReview{}
	response := v1beta1.AdmissionResponse{}
	response.Allowed = true
	response.Result = &metav1.Status{
		Message: strings.TrimSpace("Allowed if there is no envvars"),
		Code:    200,
	}

	if len(body) < 1 {
		return events.APIGatewayProxyResponse{}, ErrNameNotProvided
	}

	if err := json.Unmarshal([]byte(body), &admissionrequest); err != nil {
		log.Printf("Couldnt marshall the request %v", err)
		return events.APIGatewayProxyResponse{}, ErrNameNotProvided
	}

	response.UID = admissionrequest.Request.UID

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionrequest.Request.Resource != podResource {
		return events.APIGatewayProxyResponse{}, ErrNameNotProvided
	}

	raw := admissionrequest.Request.Object.Raw
	var RequestPod k8v1.Pod
	deserializer := scheme.Codecs.UniversalDeserializer()

	if _, _, err := deserializer.Decode(raw, nil, &RequestPod); err != nil {
		log.Printf("Couldnt deserialize request to Pod %v ", err)
		return events.APIGatewayProxyResponse{}, ErrNameNotProvided
	}

	log.Printf("Able to serialize pod %s ", RequestPod.GetObjectMeta().GetName())

	for _, container := range RequestPod.Spec.Containers {
		log.Printf("Looping through container in the pod   %s ", container.Name)
		if len(container.Env) > 1 {
			log.Printf("Container %s  has environment variables", container.Name)
			response.Allowed = false
			response.Result = &metav1.Status{
				Message: strings.TrimSpace("Has env vars so not allowing !! Allowed if there is no envvars"),
				Code:    200,
				Reason:  "Has env vars so not allowing !! Allowed if there is no envvars",
				Status:  "Has env vars so not allowing !! Allowed if there is no envvars",
			}

			admissionrequest.Response = &response
			admissionrequest.Request.Object = runtime.RawExtension{}
			admissionrequest.Request.OldObject = runtime.RawExtension{}
			resultBody, _ := json.Marshal(admissionrequest)
			return events.APIGatewayProxyResponse{
				Body:       string(resultBody[:]),
				StatusCode: 200,
			}, nil
		}

	}

	admissionrequest.Response = &response
	admissionrequest.Request.Object = runtime.RawExtension{}
	admissionrequest.Request.OldObject = runtime.RawExtension{}
	resultBody, _ := json.Marshal(admissionrequest)
	return events.APIGatewayProxyResponse{
		Body:       string(resultBody[:]),
		StatusCode: 200,
	}, nil

}

func main() {
	lambda.Start(Handler)
}
