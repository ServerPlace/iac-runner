package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
)

type Signer struct {
	secretKey string
}

func New(secret string) *Signer {
	return &Signer{
		secretKey: secret,
	}
}

func (s *Signer) Sign(object any) (string, error) {
	msg, err := GenerateCanonicalString(object, []string{"Signature"})
	if err != nil {
		return "", err
	}
	return ComputeHMAC256(msg, s.secretKey), nil
}

// GenerateCanonicalString processa uma struct e retorna sua representação canônica
func GenerateCanonicalString(obj interface{}, ignoreFields []string) (string, error) {
	val := reflect.ValueOf(obj)

	// 1. Verificação de Tipo (Detectar se é Struct)
	// Se for um ponteiro para struct, nós resolvemos o valor apontado.
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("o objeto fornecido não é uma struct (tipo: %s)", val.Kind())
	}

	typ := val.Type()
	var fieldData []string // Slice para guardar "Nome:Valor" para ordenar depois

	// 2. Iteração sobre os membros
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// SECURITY/STABILITY CHECK: Ignorar campos não exportados (letra minúscula)
		// Tentar ler um campo unexported causa panic no Go.
		if !fieldType.IsExported() {
			continue
		}

		// STRICT ENFORCEMENT: A regra solicitada é "não são ponteiros"
		// Kind() retorna a categoria do tipo (Int, String, Ptr, Struct, etc.)
		if field.Kind() == reflect.Ptr {
			continue // Pula este campo
		}
		// IgnoreFields
		if slices.Contains(ignoreFields, fieldType.Name) {
			continue
		}

		// Convertemos o valor para string de forma genérica (%v)
		strValue := fmt.Sprintf("%v", field.Interface())

		// Guardamos apenas o valor ou chave=valor?
		// Para canonicalização segura, geralmente usa-se o valor.
		// Mas como vamos ordenar por nome do campo, preciso associar os dois temporariamente.
		// A estratégia aqui é criar um map temporário ou um slice de structs auxiliares.
		// Vou usar um prefixo para ordenar e depois removo na concatenação final se necessário,
		// ou concatenamos "Valor" baseando-se na ordem das chaves.

		// Armazenando para ordenar
		entry := fmt.Sprintf("%s::%s", fieldType.Name, strValue)
		fieldData = append(fieldData, entry)
	}

	// 3. Ordenação Alfabética (Sort)
	sort.Strings(fieldData)

	// 4. Concatenação com Pipe
	// Agora extraímos apenas a parte do valor, já na ordem correta das chaves
	var values []string
	for _, entry := range fieldData {
		parts := strings.SplitN(entry, "::", 2)
		if len(parts) == 2 {
			values = append(values, parts[1])
		}
	}

	return strings.Join(values, "|"), nil
}
func ComputeHMAC256(message string, secret string) string {
	// Converta o segredo para bytes
	key := []byte(secret)

	// Inicializa o HMAC com SHA256
	h := hmac.New(sha256.New, key)

	// Escreve a mensagem no hash (Writer interface)
	h.Write([]byte(message))

	// Obtém o hash final (Sum) e codifica para Hexadecimal (padrão em IaC/Terraform)
	return hex.EncodeToString(h.Sum(nil))
}
