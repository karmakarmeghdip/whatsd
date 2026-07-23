package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"

	"whatsd/internal/types"
)

func runStatus(client *Client, args []string) error {
	return client.CallAndPrint("status", nil)
}

func runPair(client *Client, args []string) error {
	pairFlags := flag.NewFlagSet("pair", flag.ContinueOnError)
	useQR := pairFlags.Bool("qr", false, "Pair using QR code in terminal")
	phone := pairFlags.String("phone", "", "Pair using phone number (e.g., 15551234567)")
	if err := pairFlags.Parse(args); err != nil {
		return err
	}

	if *useQR {
		return handlePairQR(client)
	} else if *phone != "" {
		return handlePairPhone(client, *phone)
	}

	fmt.Println("Error: must specify either --qr or --phone <number>")
	pairFlags.Usage()
	return fmt.Errorf("invalid pair flags")
}

func handlePairQR(client *Client) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	fmt.Println("Requesting QR pairing mode...")
	req := types.Request{ID: "1", Method: "pair_qr"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	_, _ = conn.Write(data)

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp types.Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil && resp.Error != "" {
			return fmt.Errorf("daemon error: %s", resp.Error)
		}
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var event types.EventNotification
		if err := json.Unmarshal(line, &event); err == nil && event.Event == "qr" {
			dataBytes, _ := json.Marshal(event.Data)
			var qrData types.QREventData
			_ = json.Unmarshal(dataBytes, &qrData)

			if qrData.Success {
				fmt.Println("\nSuccessfully paired with WhatsApp!")
				return nil
			}
			if qrData.Err != "" {
				return fmt.Errorf("QR pairing error: %s", qrData.Err)
			}
			if qrData.Code != "" {
				fmt.Println("\nScan the QR code below using WhatsApp on your phone:")
				qrterminal.GenerateHalfBlock(qrData.Code, qrterminal.L, os.Stdout)
			}
		}
	}
	return scanner.Err()
}

func handlePairPhone(client *Client, phone string) error {
	resp, err := client.Call("pair_phone", map[string]any{"phone": phone})
	if err != nil {
		return err
	}
	resultBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(resultBytes))
	return nil
}

func runListen(client *Client, args []string) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	fmt.Println("Listening for WhatsApp daemon events (Ctrl+C to stop)...")

	req := types.Request{ID: "listen", Method: "status"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	_, _ = conn.Write(data)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var resp types.Response
		if err := json.Unmarshal(line, &resp); err == nil && resp.ID == "listen" {
			continue
		}

		var event types.EventNotification
		if err := json.Unmarshal(line, &event); err == nil && event.Event != "" {
			eventBytes, _ := json.MarshalIndent(event, "", "  ")
			fmt.Println(string(eventBytes))
		}
	}
	return scanner.Err()
}

func runLogout(client *Client, args []string) error {
	return client.CallAndPrint("logout", nil)
}
