package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Data struct {
	Nodes  []Node
	OutDir string `yaml:"out_dir" json:"out_dir"`
}

type Node struct {
	Name      string `yaml:"name"`
	IpAddress string `yaml:"ip"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Port      string `yaml:"ssh_port"`
	Type      string `yaml:"type"`
}

func (n *Node) TakeBackupC(ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	err := n.TakeBackup()
	if err != nil {
		ch <- fmt.Sprintf("%s Backup failed: %s\n", n.IpAddress, err.Error())
	} else {
		ch <- fmt.Sprintf("%s backup successful\n", n.IpAddress)
	}
}

func TakeMultipleBackup(nodes []Node) {
	ch := make(chan string)
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go n.TakeBackupC(ch, &wg)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	result := ""
	for r := range ch {
		result += r
	}

	fmt.Println(result)
}

func (n *Node) TakeBackup() error {
	commands := []string{"en", "term len 0", "show run"}
	output, err := n.ExecuteCommands(commands)
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf("%s.txt", n.Name)
	filePath := filepath.Join(OUT_DIR, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Backup output: %s\n", output)
		return err
	}
	defer file.Close()

	_, err = file.WriteString(output)
	if err != nil {
		return err
	}
	return nil
}

func (n Node) ExecuteCommands(commands []string) (string, error) {
	Ciphers := ssh.InsecureAlgorithms().Ciphers
	Ciphers = append(Ciphers, ssh.SupportedAlgorithms().Ciphers...)
	KeyExchanges := ssh.InsecureAlgorithms().KeyExchanges
	KeyExchanges = append(KeyExchanges, ssh.SupportedAlgorithms().KeyExchanges...)
	Macs := ssh.InsecureAlgorithms().MACs
	Macs = append(Macs, ssh.SupportedAlgorithms().MACs...)
	config := &ssh.ClientConfig{
		User: n.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(n.Password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echon []bool) ([]string, error) {
				// The server is prompting for a password
				if len(questions) == 1 && strings.Contains(strings.TrimSpace(strings.ToLower(questions[0])), "password:") {
					return []string{n.Password}, nil
				}
				return nil, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config: ssh.Config{
			Ciphers:      Ciphers,
			KeyExchanges: KeyExchanges,
			MACs:         Macs,
		},
	}

	client, err := ssh.Dial("tcp", n.IpAddress+":"+n.Port, config)
	if err != nil {
		msg := fmt.Sprintf("Failed to connect to host: %v on port 22, error: %v, Username: %v, Password: %v", n.IpAddress, err, n.Username, n.Password)
		return "", errors.New(msg)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		msg := fmt.Sprintf("Failed to create a session with client: %v", err.Error())
		return "", errors.New(msg)
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return "", err
	}

	output := ""

	// Start the remote shell
	if err := session.Shell(); err != nil {
		return "", err
	}

	// Goroutine to read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			output_now := scanner.Text() + "\n"
			output += output_now
		}
	}()

	// Goroutine to read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			output += scanner.Text()
		}
	}()

	// Send commands
	writer := bufio.NewWriter(stdin)

	for _, cmd := range commands {
		_, err := writer.WriteString(cmd + "\n")
		if err != nil {
			log.Printf("Error writing command: %v", err)
			break
		}
		writer.Flush()
		time.Sleep(500 * time.Millisecond)
	}

	// Close stdin to signal end of input
	stdin.Close()

	// Wait for the session to finish (optional, depending on your needs)
	session.Wait()
	return output, nil
}
