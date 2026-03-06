package controller

import (
	"testing"
)

// Estruturas auxiliares para usar nos cenários de teste
type User struct {
	Name string
	Age  int
}

type Product struct {
	ID    int
	Price float64
	Name  string // Testar ordenação (N vem depois de I, mas antes de P?)
}

type WithPointer struct {
	Title       string
	Description *string // Ponteiros devem ser ignorados segundo seu código
}

type WithPrivate struct {
	Public  string
	private string // Campos não exportados devem ser ignorados
}

type WithSignature struct {
	Data      string
	Signature string // O código hardcoded ignora o campo "Signature"
}

func TestSigner_Sign(t *testing.T) {
	// O segredo que usaremos para assinar
	secret := "my-secret-key"
	signer := New(secret)

	// Tabela de Cenários
	tests := []struct {
		name          string // Nome do caso de teste
		input         any    // O objeto a ser assinado
		expectedCan   string // A string canônica esperada (para debug/validação interna)
		expectedError bool   // Se esperamos um erro
	}{
		{
			name:          "Sucesso Simples (Ordem Alfabética)",
			input:         User{Name: "Alice", Age: 30},
			expectedCan:   "30|Alice", // Age vem antes de Name
			expectedError: false,
		},
		{
			name:          "Ordenação de Campos",
			input:         Product{ID: 1, Price: 10.5, Name: "Book"},
			expectedCan:   "1|Book|10.5", // ID, Name, Price
			expectedError: false,
		},
		{
			name:          "Ignorar Ponteiros",
			input:         WithPointer{Title: "Go Lang", Description: nil},
			expectedCan:   "Go Lang", // Description (*string) é ignorado
			expectedError: false,
		},
		{
			name:          "Ignorar Campos Privados",
			input:         WithPrivate{Public: "Visible", private: "Hidden"},
			expectedCan:   "Visible", // private é ignorado
			expectedError: false,
		},
		{
			name:          "Ignorar Campo Signature (Hardcoded no Sign)",
			input:         WithSignature{Data: "Payload", Signature: "OldHMAC"},
			expectedCan:   "Payload", // Signature é removido pela lista ignoreFields
			expectedError: false,
		},
		{
			name:          "Erro: Input não é Struct",
			input:         "Eu sou uma string, não uma struct",
			expectedCan:   "",
			expectedError: true,
		},
		{
			name:          "Erro: Input é Inteiro",
			input:         12345,
			expectedCan:   "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Executa a função Sign
			gotHMAC, err := signer.Sign(tt.input)

			// 2. Valida se o erro ocorreu conforme esperado
			if (err != nil) != tt.expectedError {
				t.Errorf("Sign() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			// Se esperamos erro e ele veio, paramos aqui (sucesso do teste de erro)
			if tt.expectedError {
				return
			}

			// 3. Validação do HMAC
			// Como calcular o HMAC na mão é chato, vamos usar a função auxiliar que
			// você já criou (ComputeHMAC256) com a string canônica que NÓS esperamos.
			// Isso garante que a lógica de "Transformar Struct -> String" está correta.
			expectedHMAC := ComputeHMAC256(tt.expectedCan, secret)

			if gotHMAC != expectedHMAC {
				t.Errorf("Sign() HMCA incorreto.\nEntrada: %+v\nCanônico Esperado: %s\nGot: %s\nWant: %s",
					tt.input, tt.expectedCan, gotHMAC, expectedHMAC)
			}
		})
	}
}

// Teste unitário isolado apenas para a função canônica (Opcional, mas útil)
func TestGenerateCanonicalString_PointerHandling(t *testing.T) {
	u := User{Name: "Bob", Age: 20}

	// Testando passar o PONTEIRO da struct ao invés da struct direta
	// Seu código tem um `val.Kind() == reflect.Ptr { val = val.Elem() }`
	// Vamos garantir que isso funciona.

	canonical, err := GenerateCanonicalString(&u, nil)
	if err != nil {
		t.Fatalf("GenerateCanonicalString falhou com ponteiro: %v", err)
	}

	expected := "20|Bob"
	if canonical != expected {
		t.Errorf("Esperado %s, recebido %s", expected, canonical)
	}
}
