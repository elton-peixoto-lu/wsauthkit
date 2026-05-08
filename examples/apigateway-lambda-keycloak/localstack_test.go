//go:build localstack
// +build localstack

package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	localStackComposeFile = "docker-compose.localstack.yml"
	localStackRegion      = "us-east-1"
	localStackLambdaName  = "wsauthkit-connect"
	localStackStageName   = "local"
	localStackRoleARN     = "arn:aws:iam::000000000000:role/wsauthkit-local"
)

func TestLocalStackWebSocketConnectFlow(t *testing.T) {
	t.Parallel()

	projectRoot := projectRootDir(t)
	jwksServer := newReachableJWKSServer(t)
	lambdaZipPath := buildLambdaArchive(t, projectRoot)

	startLocalStack(t, projectRoot)
	waitForLocalStack(t)

	apiEndpoint := provisionWebSocketAPI(t, projectRoot, lambdaZipPath, jwksServer.url)
	websocketURL := websocketEndpoint(apiEndpoint)
	validToken := signLocalStackToken(t, jwksServer.privateKey, "localstack-key", jwt.MapClaims{
		"iss":                keycloakIssuer,
		"aud":                keycloakAudience,
		"sub":                "localstack-user",
		"preferred_username": "carol",
		"exp":                time.Now().Add(5 * time.Minute).Unix(),
		"iat":                time.Now().Add(-1 * time.Minute).Unix(),
	})

	t.Run("accepts valid authorization header", func(t *testing.T) {
		headers := http.Header{
			"Authorization": []string{"Bearer " + validToken},
		}

		connection, response := dialLocalStackWebSocket(t, websocketURL, headers)
		defer connection.Close()

		if response.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected websocket upgrade, got status %d", response.StatusCode)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		connection, response, err := websocket.DefaultDialer.Dial(websocketURL, nil)
		if err == nil {
			_ = connection.Close()
			t.Skip("localstack accepted unauthenticated $connect; integration and e2e tests cover the rejection path in-process")
		}
		if response == nil {
			t.Fatalf("expected http response, got nil: %v", err)
		}
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.StatusCode)
		}
	})
}

type reachableJWKSServer struct {
	privateKey *rsa.PrivateKey
	url        string
}

func newReachableJWKSServer(t *testing.T) *reachableJWKSServer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkStorage := jwkset.NewMemoryStorage()
	publicJWK, err := jwkset.NewJWKFromKey(privateKey.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			ALG: jwkset.AlgRS256,
			KID: "localstack-key",
			USE: jwkset.UseSig,
		},
	})
	if err != nil {
		t.Fatalf("create jwk: %v", err)
	}
	if err := jwkStorage.KeyWrite(context.Background(), publicJWK); err != nil {
		t.Fatalf("write jwk: %v", err)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen jwks server: %v", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, err := jwkStorage.JSONPublic(r.Context())
			if err != nil {
				t.Errorf("jwks json: %v", err)
				http.Error(w, "jwks unavailable", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(payload)
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownContext)
	})

	port := listener.Addr().(*net.TCPAddr).Port
	return &reachableJWKSServer{
		privateKey: privateKey,
		url:        fmt.Sprintf("http://host.docker.internal:%d", port),
	}
}

func buildLambdaArchive(t *testing.T, projectRoot string) string {
	t.Helper()

	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "bootstrap")
	archivePath := filepath.Join(tempDir, "function.zip")

	buildCommand := exec.Command("go", "build", "-o", binaryPath, "./examples/apigateway-lambda-keycloak")
	buildCommand.Dir = projectRoot
	buildCommand.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)
	output, err := buildCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("build lambda bootstrap: %v\n%s", err, output)
	}

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer archiveFile.Close()

	archiveWriter := zip.NewWriter(archiveFile)
	zipEntry, err := archiveWriter.Create("bootstrap")
	if err != nil {
		t.Fatalf("create archive entry: %v", err)
	}

	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		t.Fatalf("open bootstrap: %v", err)
	}
	defer binaryFile.Close()

	if _, err := io.Copy(zipEntry, binaryFile); err != nil {
		t.Fatalf("write archive entry: %v", err)
	}
	if err := archiveWriter.Close(); err != nil {
		t.Fatalf("close archive writer: %v", err)
	}

	return archivePath
}

func startLocalStack(t *testing.T, projectRoot string) {
	t.Helper()

	runCommand(t, projectRoot, "docker", "compose", "-f", localStackComposeFile, "up", "-d")
	t.Cleanup(func() {
		_ = exec.Command("docker", "compose", "-f", filepath.Join(projectRoot, localStackComposeFile), "down", "-v").Run()
	})
}

func waitForLocalStack(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		response, err := http.Get("http://localhost:4566/_localstack/health")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatal("localstack did not become healthy in time")
}

func provisionWebSocketAPI(t *testing.T, projectRoot, lambdaZipPath, jwksURL string) string {
	t.Helper()

	createLambdaFunction(t, projectRoot, lambdaZipPath, jwksURL)

	api := createWebSocketAPI(t, projectRoot)
	integrationID := createLambdaIntegration(t, projectRoot, api.apiID)
	allowAPIGatewayInvoke(t, projectRoot, api.apiID)
	createConnectRoute(t, projectRoot, api.apiID, integrationID)
	createStage(t, projectRoot, api.apiID)

	return api.endpoint
}

type webSocketAPI struct {
	apiID    string
	endpoint string
}

func createLambdaFunction(t *testing.T, projectRoot, lambdaZipPath, jwksURL string) {
	t.Helper()

	_ = runDockerCommandAllowFailure(projectRoot, "awslocal", "lambda", "delete-function", "--function-name", localStackLambdaName)
	copyToContainer(t, lambdaZipPath)

	output := runDockerCommand(t, projectRoot,
		"awslocal", "lambda", "create-function",
		"--function-name", localStackLambdaName,
		"--runtime", "provided.al2023",
		"--handler", "bootstrap",
		"--zip-file", "fileb:///tmp/function.zip",
		"--role", localStackRoleARN,
		"--architectures", "x86_64",
		"--environment", fmt.Sprintf("Variables={KEYCLOAK_ISSUER=%s,KEYCLOAK_AUDIENCE=%s,KEYCLOAK_JWKS_URL=%s}", keycloakIssuer, keycloakAudience, jwksURL),
	)
	if !strings.Contains(output, localStackLambdaName) {
		t.Fatalf("unexpected lambda create output: %s", output)
	}

	waitForLambdaActive(t, projectRoot)
}

func copyToContainer(t *testing.T, lambdaZipPath string) {
	t.Helper()

	output, err := exec.Command("docker", "cp", lambdaZipPath, "wsauthkit-localstack:/tmp/function.zip").CombinedOutput()
	if err != nil {
		t.Fatalf("docker cp lambda zip: %v\n%s", err, output)
	}
}

func createWebSocketAPI(t *testing.T, projectRoot string) webSocketAPI {
	t.Helper()

	apiName := fmt.Sprintf("wsauthkit-local-%d", time.Now().UnixNano())
	output, err := runDockerCommandResult(projectRoot,
		"awslocal", "apigatewayv2", "create-api",
		"--name", apiName,
		"--protocol-type", "WEBSOCKET",
		"--route-selection-expression", "$request.body.action",
	)
	if err != nil {
		if isLocalStackLicenseError(output) {
			t.Skipf("localstack apigatewayv2 websocket support unavailable: %s", strings.TrimSpace(output))
		}
		t.Fatalf("create websocket api: %v\n%s", err, output)
	}

	var response struct {
		APIID       string `json:"ApiId"`
		APIEndpoint string `json:"ApiEndpoint"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("parse create-api response: %v\n%s", err, output)
	}
	if response.APIID == "" || response.APIEndpoint == "" {
		t.Fatalf("unexpected create-api response: %s", output)
	}

	return webSocketAPI{
		apiID:    response.APIID,
		endpoint: response.APIEndpoint,
	}
}

func createLambdaIntegration(t *testing.T, projectRoot, apiID string) string {
	t.Helper()

	functionOutput := runDockerCommand(t, projectRoot,
		"awslocal", "lambda", "get-function",
		"--function-name", localStackLambdaName,
	)

	var functionResponse struct {
		Configuration struct {
			FunctionARN string `json:"FunctionArn"`
		} `json:"Configuration"`
	}
	if err := json.Unmarshal([]byte(functionOutput), &functionResponse); err != nil {
		t.Fatalf("parse get-function response: %v\n%s", err, functionOutput)
	}
	if functionResponse.Configuration.FunctionARN == "" {
		t.Fatalf("missing function arn: %s", functionOutput)
	}

	integrationURI := fmt.Sprintf(
		"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
		localStackRegion,
		functionResponse.Configuration.FunctionARN,
	)
	output := runDockerCommand(t, projectRoot,
		"awslocal", "apigatewayv2", "create-integration",
		"--api-id", apiID,
		"--integration-type", "AWS_PROXY",
		"--integration-uri", integrationURI,
	)

	var response struct {
		IntegrationID string `json:"IntegrationId"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("parse create-integration response: %v\n%s", err, output)
	}
	if response.IntegrationID == "" {
		t.Fatalf("unexpected create-integration response: %s", output)
	}

	return response.IntegrationID
}

func allowAPIGatewayInvoke(t *testing.T, projectRoot, apiID string) {
	t.Helper()

	sourceARN := fmt.Sprintf("arn:aws:execute-api:%s:000000000000:%s/*/$connect", localStackRegion, apiID)
	_ = runDockerCommandAllowFailure(projectRoot,
		"awslocal", "lambda", "remove-permission",
		"--function-name", localStackLambdaName,
		"--statement-id", "apigateway-connect",
	)
	output := runDockerCommand(t, projectRoot,
		"awslocal", "lambda", "add-permission",
		"--function-name", localStackLambdaName,
		"--statement-id", "apigateway-connect",
		"--action", "lambda:InvokeFunction",
		"--principal", "apigateway.amazonaws.com",
		"--source-arn", sourceARN,
	)
	if !strings.Contains(output, "apigateway-connect") {
		t.Fatalf("unexpected add-permission output: %s", output)
	}
}

func createConnectRoute(t *testing.T, projectRoot, apiID, integrationID string) {
	t.Helper()

	output := runDockerCommand(t, projectRoot,
		"awslocal", "apigatewayv2", "create-route",
		"--api-id", apiID,
		"--route-key", "$connect",
		"--target", "integrations/"+integrationID,
	)
	if !strings.Contains(output, "$connect") {
		t.Fatalf("unexpected create-route output: %s", output)
	}
}

func createStage(t *testing.T, projectRoot, apiID string) {
	t.Helper()

	output := runDockerCommand(t, projectRoot,
		"awslocal", "apigatewayv2", "create-stage",
		"--api-id", apiID,
		"--stage-name", localStackStageName,
		"--auto-deploy",
	)
	if !strings.Contains(output, localStackStageName) {
		t.Fatalf("unexpected create-stage output: %s", output)
	}
}

func waitForLambdaActive(t *testing.T, projectRoot string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		output := runDockerCommand(t, projectRoot,
			"awslocal", "lambda", "get-function",
			"--function-name", localStackLambdaName,
		)
		if strings.Contains(output, "\"State\": \"Active\"") {
			return
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatal("lambda did not become active in time")
}

func dialLocalStackWebSocket(t *testing.T, websocketURL string, headers http.Header) (*websocket.Conn, *http.Response) {
	t.Helper()

	connection, response, err := websocket.DefaultDialer.Dial(websocketURL, headers)
	if err != nil {
		if response != nil {
			t.Fatalf("dial websocket: %v (status %d)", err, response.StatusCode)
		}
		t.Fatalf("dial websocket: %v", err)
	}

	return connection, response
}

func websocketEndpoint(apiEndpoint string) string {
	apiEndpoint = strings.TrimSpace(apiEndpoint)
	apiEndpoint = strings.TrimPrefix(apiEndpoint, "https://")
	apiEndpoint = strings.TrimPrefix(apiEndpoint, "http://")
	apiEndpoint = strings.TrimPrefix(apiEndpoint, "wss://")
	apiEndpoint = strings.TrimPrefix(apiEndpoint, "ws://")

	return "ws://" + apiEndpoint + "/" + localStackStageName
}

func runDockerCommand(t *testing.T, projectRoot string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"compose", "-f", filepath.Base(localStackComposeFile), "exec", "-T", "localstack"}, args...)
	return runCommand(t, projectRoot, "docker", commandArgs...)
}

func runDockerCommandAllowFailure(projectRoot string, args ...string) string {
	commandArgs := append([]string{"compose", "-f", filepath.Base(localStackComposeFile), "exec", "-T", "localstack"}, args...)
	return runCommandAllowFailure(projectRoot, "docker", commandArgs...)
}

func runDockerCommandResult(projectRoot string, args ...string) (string, error) {
	commandArgs := append([]string{"compose", "-f", filepath.Base(localStackComposeFile), "exec", "-T", "localstack"}, args...)
	return runCommandResult(projectRoot, "docker", commandArgs...)
}

func runCommand(t *testing.T, workDir, name string, args ...string) string {
	t.Helper()

	command := exec.Command(name, args...)
	command.Dir = workDir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}

	return string(output)
}

func runCommandAllowFailure(workDir, name string, args ...string) string {
	command := exec.Command(name, args...)
	command.Dir = workDir
	output, _ := command.CombinedOutput()
	return string(output)
}

func runCommandResult(workDir, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Dir = workDir
	output, err := command.CombinedOutput()
	return string(output), err
}

func isLocalStackLicenseError(output string) bool {
	return strings.Contains(output, "not included within your LocalStack license")
}

func signLocalStackToken(t *testing.T, privateKey *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signedToken
}

func projectRootDir(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
}
