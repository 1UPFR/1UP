package nyuu

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/1UPFR/1UP/internal/config"
)

type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func TestConnection(cfg *config.NyuuConfig) *TestResult {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var conn net.Conn
	var err error

	if cfg.SSL {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("Connexion impossible : %v", err)}
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Lire le banner
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("Pas de reponse du serveur : %v", err)}
	}
	banner := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(banner, "200") && !strings.HasPrefix(banner, "201") {
		return &TestResult{Success: false, Message: fmt.Sprintf("Reponse inattendue : %s", banner)}
	}

	// Auth
	if cfg.User != "" {
		fmt.Fprintf(conn, "AUTHINFO USER %s\r\n", cfg.User)
		n, err = conn.Read(buf)
		if err != nil {
			return &TestResult{Success: false, Message: "Erreur envoi utilisateur"}
		}
		resp := strings.TrimSpace(string(buf[:n]))

		if strings.HasPrefix(resp, "381") {
			fmt.Fprintf(conn, "AUTHINFO PASS %s\r\n", cfg.Password)
			n, err = conn.Read(buf)
			if err != nil {
				return &TestResult{Success: false, Message: "Erreur envoi mot de passe"}
			}
			resp = strings.TrimSpace(string(buf[:n]))
		}

		if !strings.HasPrefix(resp, "281") {
			return &TestResult{Success: false, Message: fmt.Sprintf("Authentification echouee : %s", resp)}
		}
	}

	fmt.Fprintf(conn, "QUIT\r\n")

	ssl := ""
	if cfg.SSL {
		ssl = " (SSL)"
	}
	return &TestResult{Success: true, Message: fmt.Sprintf("Connecte a %s%s - Authentifie", addr, ssl)}
}
