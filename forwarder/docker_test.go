package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	EnvUseSudo                      = "USE_SUDO"
	EnvPortforwardImage             = "PORTFORWARD_IMAGE"
	DockerTestContainersLabel       = "portforward-golang-test"
	DockerTestNetworkName           = "portforward-golang-test-network"
	DockerTestTorproxyContainerName = "portforward-golang-test-torproxy"
	DockerTestTorproxyPort          = 9050
	RegexIpv4                       = "^(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$"
	DockerDeployThinktime           = 2 * time.Second
	HttpTimeout                     = 5 * time.Second
	HttpRetries                     = 3
)

var (
	portForwardImage string
	runSudo          bool
)

type PortRangeForward struct {
	LocalPortStart  int64
	LocalPortEnd    int64
	RemoteHost      string
	RemotePortStart int64
	RemotePortEnd   int64
}

func dockertestSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	allEnv := getAllEnvironmentVariables()
	portForwardImage = allEnv[EnvPortforwardImage]
	if allEnv[EnvUseSudo] == "1" {
		runSudo = true
	}

	if portForwardImage == "" {
		panic(fmt.Errorf("no %s defined", EnvPortforwardImage))
	}

	clearTestContainers()
	clearTestNetwork()

	err := createTestNetwork()
	if err != nil {
		panic(err)
	}
}

func dockertestTeardown() {
	clearTestContainers()
	clearTestNetwork()
}

func runCmd(chunks ...string) (string, error) {
	if runSudo {
		newChunks := []string{"sudo"}
		newChunks = append(newChunks, chunks...)
		chunks = newChunks
	}

	fmt.Println("Running command", chunks, "...")
	outputBytes, err := exec.Command(chunks[0], chunks[1:]...).Output()
	output := string(outputBytes)

	if err != nil {
		fmt.Println("Command failed!", err)
	}
	if output != "" {
		fmt.Println("Command output:", strings.TrimSpace(output))
	} else {
		fmt.Println("No command output")
	}

	return output, err
}

func createTestNetwork() error {
	_, err := runCmd("docker", "network", "create", DockerTestNetworkName)
	return err
}

func clearTestNetwork() {
	_, _ = runCmd("docker", "network", "rm", DockerTestNetworkName)
}

func clearTestContainers() {
	output, _ := runCmd("docker", "ps", "-q", "--filter", fmt.Sprintf("label=%s", DockerTestContainersLabel))
	if output == "" {
		return
	}

	containers := strings.Split(output, "\n")
	cmds := []string{"stop", "rm"}
	for _, cmd := range cmds {
		finalCmd := []string{"docker", "container", cmd}
		finalCmd = append(finalCmd, containers...)
		_, _ = runCmd(finalCmd...)
	}
}

func runContainer(name string, args ...string) error {
	cmd := []string{"docker", "run", "-d", "--rm", "--name", name, "--label", DockerTestContainersLabel, "--network", DockerTestNetworkName}
	cmd = append(cmd, args...)
	_, err := runCmd(cmd...)
	return err
}

func startNginxContainer(name string) error {
	args := []string{"--hostname", name, "nginxdemos/hello:plain-text"}
	return runContainer(name, args...)
}

// startTorProxyContainer starts a test container running tor-privoxy
func startTorProxyContainer() error {
	return runContainer(DockerTestTorproxyContainerName, "dperson/torproxy")
}

func startPortForwardContainer(name string, ports []PortForward, portsRanges []PortRangeForward, proxy *SocksProxy, exposePorts bool) error {
	var args []string
	for i, port := range ports {
		envName := fmt.Sprintf("PORT%d", i)
		envValue := fmt.Sprintf("%d:%s:%d", port.LocalPort, port.RemoteHost, port.RemotePort)
		envArg := fmt.Sprintf("%s=%s", envName, envValue)
		args = append(args, "-e", envArg)

		if exposePorts {
			portArg := fmt.Sprintf("%d:%d", port.LocalPort, port.LocalPort)
			args = append(args, "-p", portArg)
		}
	}

	for i, port := range portsRanges {
		envName := fmt.Sprintf("PORTRNG%d", i)
		envValue := fmt.Sprintf("%d-%d:%s:%d-%d", port.LocalPortStart, port.LocalPortEnd, port.RemoteHost, port.RemotePortStart, port.RemotePortEnd)
		envArg := fmt.Sprintf("%s=%s", envName, envValue)
		portArg := fmt.Sprintf("%d-%d:%d-%d", port.LocalPortStart, port.LocalPortEnd, port.LocalPortStart, port.LocalPortEnd)

		portArgs := []string{"-p", portArg, "-e", envArg}
		args = append(args, portArgs...)
	}

	if proxy != nil {
		args = append(args, "-e", fmt.Sprintf("SOCKS_PROXY=%s:%d", proxy.Host, proxy.Port))
	}

	args = append(args, portForwardImage)
	return runContainer(name, args...)
}

func requestNginxGetHostname(port int64) (string, error) {
	url := fmt.Sprintf("http://localhost:%d", port)
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}

	responseBodyByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	responseBody := string(responseBodyByte)
	linePrefix := "Server name: "
	for _, line := range strings.Split(responseBody, "\n") {
		// Expected something like: "Server name: nginx1"
		if strings.HasPrefix(line, linePrefix) {
			hostname := strings.Replace(line, linePrefix, "", 1)
			return hostname, nil
		}
	}

	return "", nil
}

func requestHttpbinIP(baseURL string) (string, error) {
	url := fmt.Sprintf("%s/ip", baseURL)
	client := http.Client{Timeout: HttpTimeout}

	var response *http.Response
	var err error
	for i := 0; i < HttpRetries; i++ {
		response, err = client.Get(url)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}

	responseBodyByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var responseBody map[string]string
	err = json.Unmarshal(responseBodyByte, &responseBody)
	if err != nil {
		return "", err
	}

	return responseBody["origin"], nil
}

func TestSimpleForwardIntegration(t *testing.T) {
	dockertestSetup(t)
	defer dockertestTeardown()

	portsForwarded := []PortForward{
		{
			LocalPort:  8881,
			RemoteHost: "nginx1",
			RemotePort: 80,
		},
		{
			LocalPort:  8882,
			RemoteHost: "nginx2",
			RemotePort: 80,
		},
	}
	var expectedHostnames []string

	for _, portForward := range portsForwarded {
		name := portForward.RemoteHost
		expectedHostnames = append(expectedHostnames, name)

		err := startNginxContainer(name)
		if err != nil {
			t.Error("failed starting nginx container", err)
			return
		}
	}

	err := startPortForwardContainer("portforward-nginxs", portsForwarded, nil, nil, true)
	if err != nil {
		t.Error("failed starting port forward container", err)
		return
	}

	time.Sleep(DockerDeployThinktime)

	for i, portForward := range portsForwarded {
		expectedHostname := expectedHostnames[i]
		responseHostname, err := requestNginxGetHostname(portForward.LocalPort)
		assert.Nil(t, err)
		assert.Equal(t, expectedHostname, responseHostname)
	}
}

// TestProxyForwardIntegration
// 1. Run a Tor proxy container
// 2. Run a portforward container, redirecting a port to httpbin.org:80, using the Tor proxy
// 3. Request the current public IP to httpbin.org/ip
// 4. Request the current public IP to localhost:forwardport/ip (going through the portforward and proxy)
// 5. Compare the IPs. Must be valid and not match
func TestProxyForwardIntegration(t *testing.T) {
	dockertestSetup(t)
	defer dockertestTeardown()

	err := startTorProxyContainer()
	if err != nil {
		t.Error("failed starting tor proxy container")
		return
	}

	localPort := int64(8080)
	remoteHost := "httpbin.org"
	portForward := PortForward{
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: 80,
	}
	proxy := SocksProxy{
		Host: DockerTestTorproxyContainerName,
		Port: DockerTestTorproxyPort,
	}

	err = startPortForwardContainer("portforward-httpbin", []PortForward{portForward}, nil, &proxy, true)
	if err != nil {
		t.Error("failed starting portforward container", err)
		return
	}

	time.Sleep(DockerDeployThinktime)

	ipVanilla, err := requestHttpbinIP(fmt.Sprintf("https://%s", remoteHost))
	if err != nil {
		t.Error("failed vanilla httpbin request", err)
		return
	}

	ipPortForward, err := requestHttpbinIP(fmt.Sprintf("http://localhost:%d", localPort))
	assert.Nil(t, err)

	fmt.Printf("Acquired Public IPs: local=%s portForward=%s\n", ipVanilla, ipPortForward)
	assert.NotEmpty(t, ipVanilla)
	assert.NotEmpty(t, ipPortForward)
	assert.Regexp(t, RegexIpv4, ipVanilla)
	assert.Regexp(t, RegexIpv4, ipPortForward)
	assert.NotEqual(t, ipVanilla, ipPortForward)
}

// TestSimpleRangeForwardIntegration
// 1. Run N nginx containers
// 2. Run a "middle forwarder", forwarding each container
// 3. Run a "final forwarder" (the currently tested), with a range forward to the "middle forwarder"
// 4. Test each port on the "final forwarder"
func TestSimpleRangeForwardIntegration(t *testing.T) {
	dockertestSetup(t)
	defer dockertestTeardown()

	var portStart int64 = 8880
	var portEnd int64 = 8889
	portRangeLength := portEnd - portStart

	var middleForwarderPorts []PortForward
	var i int64
	for i = 0; i < portRangeLength; i++ {
		containerName := fmt.Sprintf("nginx%d", i)
		currentPort := portStart + i

		portForward := PortForward{
			LocalPort:  currentPort,
			RemoteHost: containerName,
			RemotePort: 80,
		}
		middleForwarderPorts = append(middleForwarderPorts, portForward)

		err := startNginxContainer(containerName)
		if err != nil {
			t.Error("failed starting nginx container", i, err)
			return
		}
	}

	middleForwarderName := "forwarder-middle"
	err := startPortForwardContainer(middleForwarderName, middleForwarderPorts, nil, nil, false)
	if err != nil {
		t.Error("failed starting middle port forward container", err)
		return
	}

	forwardRange := PortRangeForward{
		LocalPortStart:  portStart,
		LocalPortEnd:    portEnd,
		RemoteHost:      middleForwarderName,
		RemotePortStart: portStart,
		RemotePortEnd:   portEnd,
	}

	err = startPortForwardContainer("forwarder-final", nil, []PortRangeForward{forwardRange}, nil, true)
	if err != nil {
		t.Error("failed starting final port forward container", err)
		return
	}

	time.Sleep(DockerDeployThinktime)

	for _, portForward := range middleForwarderPorts {
		expectedHostname := portForward.RemoteHost
		responseHostname, err := requestNginxGetHostname(portForward.LocalPort)
		assert.Nil(t, err)
		assert.Equal(t, expectedHostname, responseHostname)
	}
}
