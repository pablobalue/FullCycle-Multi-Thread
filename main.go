package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Address struct {
	CEP          string `json:"cep"`
	Street       string `json:"street,omitempty"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
}

type BrasilAPIResponse struct {
	CEP          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
}

type ViaCEPResponse struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
}

type APIResult struct {
	Addr   Address
	Source string
	Err    error
}

func fetchBrasilAPI(ctx context.Context, cep string, ch chan<- APIResult) {
	url := fmt.Sprintf("https://brasilapi.com.br/api/cep/v1/%s", cep)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		ch <- APIResult{Err: err, Source: "BrasilAPI"}
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- APIResult{Err: err, Source: "BrasilAPI"}
		return
	}
	defer resp.Body.Close()

	var r BrasilAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		ch <- APIResult{Err: err, Source: "BrasilAPI"}
		return
	}

	ch <- APIResult{
		Addr: Address{
			CEP:          r.CEP,
			Street:       r.Street,
			Neighborhood: r.Neighborhood,
			City:         r.City,
			State:        r.State,
		},
		Source: "BrasilAPI",
	}
}

func fetchViaCEP(ctx context.Context, cep string, ch chan<- APIResult) {
	url := fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		ch <- APIResult{Err: err, Source: "ViaCEP"}
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- APIResult{Err: err, Source: "ViaCEP"}
		return
	}
	defer resp.Body.Close()

	var v ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		ch <- APIResult{Err: err, Source: "ViaCEP"}
		return
	}

	ch <- APIResult{
		Addr: Address{
			CEP:          v.CEP,
			Street:       v.Logradouro,
			Complement:   v.Complemento,
			Neighborhood: v.Bairro,
			City:         v.Localidade,
			State:        v.UF,
		},
		Source: "ViaCEP",
	}
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Uso: go run main.go <cep>")
		os.Exit(1)
	}
	cep := os.Args[1]

	// timeout de 1 segundo
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch := make(chan APIResult, 2)
	go fetchBrasilAPI(ctx, cep, ch)
	go fetchViaCEP(ctx, cep, ch)

	select {
	case res := <-ch:
		// Cancela a requisição mais lenta
		cancel()
		if res.Err != nil {
			fmt.Printf("Erro ao buscar CEP: %v\n", res.Err)
			os.Exit(1)
		}
		fmt.Printf("Resposta da %s:\n", res.Source)
		fmt.Printf("CEP: %s\nRua: %s\nBairro: %s\nCidade: %s\nEstado: %s\n",
			res.Addr.CEP,
			res.Addr.Street,
			res.Addr.Neighborhood,
			res.Addr.City,
			res.Addr.State,
		)
	case <-ctx.Done():
		// Se nenhuma resposta for recebida dentro do timeout
		fmt.Println("Timeout de 1 segundo excedido")
		os.Exit(1)
	}
}
