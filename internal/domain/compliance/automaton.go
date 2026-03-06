package compliance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ServerPlace/iac-runner/pkg/log"
)

// --- 1. DEFINIÇÃO DO ALFABETO (TOKENS) ---
// O Lexer converte diretórios "sujos" nestes símbolos limpos.
type TokenType int

const (
	TokenNone TokenType = iota // Diretório comum
	TokenEnv
	TokenRegion // Tem env.hcl E region.hcl
	TokenFolder // Tem folder.hcl
	TokenRoot   // Tem root.hcl
)

func (t TokenType) String() string {
	switch t {
	case TokenNone:
		return "None"
	case TokenRegion:
		return "Region (region.hcl)"
	case TokenEnv:
		return "Env (env.hcl)"
	case TokenFolder:
		return "Folder (folder.hcl)"
	case TokenRoot:
		return "Root (root.hcl)"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

const (
	InvalidHierarchyMessage string = "found '%s' while expecting '%s'"
)

// --- 2. DEFINIÇÃO DOS ESTADOS ---
type State int

const (
	StateStart       State = iota // Começou na Stack
	StateFoundRegion              // Region OK Lvl 1
	StateFoundEnv                 // Env OK Lvl 2
	StateFoundFolder              // Folder OK lvl 3
	StateValid                    // Root Ok. Accept
	StateError                    // Invalid. Reject
)

// --- 3. O LEXER (Scanner de IO) ---
// Função pura que analisa UM diretório e retorna seu Token
func scanDirectory(dir string) TokenType {
	hasRoot := fileExists(filepath.Join(dir, "root.hcl"))
	hasFolder := fileExists(filepath.Join(dir, "folder.hcl"))
	hasEnv := fileExists(filepath.Join(dir, "env.hcl"))
	hasRegion := fileExists(filepath.Join(dir, "region.hcl"))

	// A ordem dos ifs define a precedência se um dir tiver múltiplos arquivos (caso de borda)
	if hasRoot {
		return TokenRoot
	}
	if hasFolder {
		return TokenFolder
	}
	if hasRegion {
		return TokenRegion
	}
	if hasEnv {
		return TokenEnv
	}
	return TokenNone
}

// --- 4. O PARSER (Máquina de Estados) ---
// Aplica a Função de Transição: f(EstadoAtual, Token) -> NovoEstado
func transition(current State, token TokenType) (State, error) {
	// 1. REGRA GLOBAL: Ignorar diretórios vazios (sem arquivos .hcl de estrutura)
	// Isso permite navegar por pastas intermediárias (ex: "platform/", "hub-network/")
	// sem quebrar a máquina de estados. Mantém o estado atual e continua subindo.
	if token == TokenNone {
		return current, nil
	}

	switch current {

	case StateStart:
		// Estado inicial: Estamos na stack, procurando a Region acima
		if token == TokenRegion {
			return StateFoundRegion, nil
		}
		// Se achar Env ou Folder direto, pulou a Region -> Erro
		return StateStart, fmt.Errorf(InvalidHierarchyMessage, token.String(), TokenRegion.String())

	case StateFoundRegion:
		// Achamos a Region. O próximo nível OBRIGATÓRIO é o Env.
		if token == TokenEnv {
			return StateFoundEnv, nil
		}
		// Se achou Folder direto, pulou o Env -> Erro
		return StateStart, fmt.Errorf(InvalidHierarchyMessage, token.String(), TokenEnv.String())

	case StateFoundEnv:
		// Achamos o Env. O próximo nível OBRIGATÓRIO é o Folder.
		if token == TokenFolder {
			return StateFoundFolder, nil
		}
		// Se achou Root direto, pulou o Folder -> Erro
		return StateStart, fmt.Errorf(InvalidHierarchyMessage, token.String(), TokenFolder.String())

	case StateFoundFolder:
		// Achamos o Folder. O próximo nível é o Root.
		if token == TokenRoot {
			return StateValid, nil
		}
		// Permitir aninhamento de folders (Folder dentro de Folder)
		// Se achar outro folder, mantém no estado FoundFolder
		if token == TokenFolder {
			return StateFoundFolder, nil
		}
		return StateStart, fmt.Errorf(InvalidHierarchyMessage, token.String(), TokenRoot.String())

	case StateValid:
		// Já validou o Root. Qualquer coisa acima é ignorada.
		return StateValid, nil

	default:
		return StateError, fmt.Errorf("estado desconhecido na máquina de estados")
	}
}

// --- 5. RUNNER (A Máquina) ---
// initialState define o ponto de entrada na máquina de estados.
// Permite ao caller (rules.go) decidir o nível a partir do qual validar,
// com base em dados do registry (ex: componente de organização → StateFoundEnv).
func ValidateStackHierarchy(ctx context.Context, stackPath string, initialState State) error {
	absPath, _ := filepath.Abs(stackPath)
	currentDir := filepath.Dir(absPath)

	state := initialState
	// Folder creation: stack dentro de diretório chamado "folder" não requer region/env.
	// Exceção baseada em convenção de path — aplicada independentemente do initialState.
	if filepath.Base(currentDir) == "folder" && state == StateStart {
		state = StateFoundEnv
	}

	logger := log.FromContext(ctx)

	logger.Debug().Msgf("Validating stack hierarchy in %s", currentDir)

	// Loop de Travessia
	for {
		logger.Debug().Msgf("Scanning %s in state: %v", currentDir, state)
		// A. Lê o Token do disco
		token := scanDirectory(currentDir)

		// B. Calcula o próximo estado
		nextState, err := transition(state, token)
		if err != nil {
			return fmt.Errorf("violação de topologia em '%s': %w", currentDir, err)
		}
		state = nextState

		// C. Aceitação Imediata
		if state == StateValid {
			return nil
		}

		// D. Movimento (Sobe um nível)
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Chegou no / e não terminou
			break
		}
		currentDir = parentDir
	}

	// Diagnóstico final se acabar a fita
	return explainMissingState(state)
}

func explainMissingState(s State) error {
	switch s {
	case StateStart:
		return fmt.Errorf("hierarquia incompleta: nunca encontrou 'region.hcl'")
	case StateFoundRegion:
		return fmt.Errorf("hierarquia incompleta: encontrou region, mas falta 'env.hcl' acima dela")
	case StateFoundEnv:
		return fmt.Errorf("hierarquia incompleta: encontrou env, mas falta 'folder.hcl' acima dele")
	case StateFoundFolder:
		return fmt.Errorf("hierarquia incompleta: encontrou folder, mas falta 'root.hcl' na raiz")
	default:
		return fmt.Errorf("erro desconhecido")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && !info.IsDir()
}
