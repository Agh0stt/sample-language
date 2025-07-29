package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ====== Data Storage ======
var mutables = map[string]interface{}{}
var constants = map[string]interface{}{}

type Function struct {
	params []string
	body   []string
}

var functions = map[string]Function{}

// ====== Helpers ======
func interpolate(s string) string {
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		varName := re.FindStringSubmatch(match)[1]
		if val, ok := mutables[varName]; ok {
			return fmt.Sprint(val)
		}
		if val, ok := constants[varName]; ok {
			return fmt.Sprint(val)
		}
		return ""
	})
}

func evalExpr(expr string) interface{} {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return expr[1 : len(expr)-1]
	}
	if val, ok := mutables[expr]; ok {
		return val
	}
	if val, ok := constants[expr]; ok {
		return val
	}
	if i, err := strconv.Atoi(expr); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f
	}
	return expr
}

func tokenize(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{}
	}
	return strings.Fields(line)
}

// ====== Execution ======
func execBlock(lines []string, index *int) {
	for *index < len(lines) {
		tokens := tokenize(lines[*index])
		if len(tokens) == 0 {
			*index++
			continue
		}
		cmd := tokens[0]

		// End of block
		if cmd == "end" {
			return
		}

		// ===== Print =====
		if cmd == "print" {
			text := strings.Join(tokens[1:], " ")
			if strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") {
				text = text[1 : len(text)-1]
			}
			out := interpolate(text)
			out = strings.ReplaceAll(out, `\n`, "\n")
			out = strings.ReplaceAll(out, `\t`, "\t")
			fmt.Println(out)
		}

		// ===== Variables =====
		if cmd == "let" && len(tokens) >= 4 && tokens[2] == "=" {
			mutables[tokens[1]] = evalExpr(tokens[3])
		}
		if cmd == "const" && len(tokens) >= 4 && tokens[2] == "=" {
			constants[tokens[1]] = evalExpr(tokens[3])
		}
  
// ===== Input (Type Aware) =====
if cmd == "input" && len(tokens) >= 3 {
    varName := tokens[1]
    prompt := strings.Join(tokens[2:], " ")
    if strings.HasPrefix(prompt, "\"") && strings.HasSuffix(prompt, "\"") {
        prompt = prompt[1 : len(prompt)-1]
    }
    fmt.Print(prompt)
    reader := bufio.NewReader(os.Stdin)
    text, _ := reader.ReadString('\n')
    text = strings.TrimSpace(text)

    // Type detection
    if i, err := strconv.Atoi(text); err == nil {
        mutables[varName] = i
    } else if f, err := strconv.ParseFloat(text, 64); err == nil {
        mutables[varName] = f
    } else {
        mutables[varName] = text
    }
}            // ===== Classes =====
if cmd == "class" {
	className := tokens[1]
	methods := map[string]Function{}
	*index++
	for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
		tok := tokenize(lines[*index])
		if len(tok) > 0 && tok[0] == "func" {
			mName := tok[1]
			params := []string{}
			if len(tok) > 2 {
				for _, p := range tok[2:] {
					if p != ":" && p != "," {
						params = append(params, p)
					}
				}
			}
			*index++
			body := []string{}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				body = append(body, lines[*index])
				*index++
			}
			methods[mName] = Function{params, body}
		}
		*index++
	}
	functions["__class_"+className] = Function{[]string{}, []string{}} // placeholder
	mutables["__methods_"+className] = methods
}

// ===== New Instance =====
if cmd == "new" && len(tokens) >= 2 {
	className := tokens[1]
	args := tokens[2:]
	instName := "obj" + className + strconv.Itoa(len(mutables)) // unique
	mutables[instName] = className

	// Call init if exists
	methods := mutables["__methods_"+className].(map[string]Function)
	if initFunc, ok := methods["init"]; ok {
		backup := make(map[string]interface{})
		for k, v := range mutables {
			backup[k] = v
		}
		for i, p := range initFunc.params {
			mutables["self_"+p] = evalExpr(args[i])
		}
		idx := 0
		execBlock(initFunc.body, &idx)
		mutables = backup
	}
	mutables["last_instance"] = instName
}

// ===== Method Call =====
if strings.Contains(cmd, "_") {
	parts := strings.SplitN(cmd, "_", 2)
	inst := parts[0]
	method := parts[1]
	className := mutables[inst].(string)
	methods := mutables["__methods_"+className].(map[string]Function)

	if fn, ok := methods[method]; ok {
		backup := make(map[string]interface{})
		for k, v := range mutables {
			backup[k] = v
		}
		idx := 0
		execBlock(fn.body, &idx)
		mutables = backup
	}
}
		// ===== Sleep =====
		if cmd == "sleep" && len(tokens) >= 2 {
			secs, err := strconv.ParseFloat(tokens[1], 64)
			if err == nil {
				ms := time.Duration(secs * float64(time.Second))
				time.Sleep(ms)
			} else {
				fmt.Println("Invalid sleep time")
			}
		}

		// ===== If / Elif / Else =====
		if cmd == "if" {
			cond := evalExpr(tokens[1]).(int)
			*index++
			if cond != 0 {
				execBlock(lines, index)
			} else {
				for *index < len(lines) {
					next := tokenize(lines[*index])
					if len(next) > 0 && next[0] == "elif" {
						cond2 := evalExpr(next[1]).(int)
						*index++
						if cond2 != 0 {
							execBlock(lines, index)
							break
						}
					} else if len(next) > 0 && next[0] == "else" {
						*index++
						execBlock(lines, index)
						break
					} else if len(next) > 0 && next[0] == "end" {
						break
					} else {
						*index++
					}
				}
			}
		}

		// ===== Switch =====
		if cmd == "switch" {
			switchVal := evalExpr(tokens[1])
			match := false
			*index++
			for *index < len(lines) {
				next := tokenize(lines[*index])
				if len(next) > 0 && next[0] == "case" {
					caseVal := evalExpr(next[1])
					if caseVal == switchVal || match {
						match = true
						*index++
						execBlock(lines, index)
					} else {
						for *index < len(lines) {
							n := tokenize(lines[*index])
							if len(n) > 0 && (n[0] == "case" || n[0] == "default" || n[0] == "end") {
								break
							}
							*index++
						}
					}
				} else if len(next) > 0 && next[0] == "default" {
					*index++
					execBlock(lines, index)
				} else if len(next) > 0 && next[0] == "end" {
					break
				} else {
					*index++
				}
			}
		}

		// ===== For Loop =====
		if cmd == "for" && len(tokens) >= 6 && tokens[2] == "=" && tokens[4] == "to" {
			start := evalExpr(tokens[3]).(int)
			end := evalExpr(tokens[5]).(int)
			for i := start; i <= end; i++ {
				mutables[tokens[1]] = i
				innerIndex := *index + 1
				execBlock(lines, &innerIndex)
			}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				*index++
			}
		}

		// ===== Repeat Loop =====
		if cmd == "repeat" && len(tokens) >= 2 {
			count, err := strconv.Atoi(tokens[1])
			if err != nil {
				fmt.Println("Invalid repeat count:", tokens[1])
			} else {
				for i := 0; i < count; i++ {
					innerIndex := *index + 1
					execBlock(lines, &innerIndex)
				}
			}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				*index++
			}
		}

		// ===== While Loop =====
		if cmd == "while" && len(tokens) >= 2 {
			condExpr := tokens[1]
			for evalExpr(condExpr).(int) != 0 {
				innerIndex := *index + 1
				execBlock(lines, &innerIndex)
			}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				*index++
			}
		}

		// ===== LoopUntil =====
		if cmd == "loopuntil" && len(tokens) >= 2 {
			condExpr := tokens[1]
			for {
				innerIndex := *index + 1
				execBlock(lines, &innerIndex)
				if evalExpr(condExpr).(int) != 0 {
					break
				}
			}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				*index++
			}
		}

		// ===== Func Define =====
		if cmd == "func" {
			name := tokens[1]
			params := []string{}
			if len(tokens) > 2 {
				for _, p := range tokens[2:] {
					if p != ":" && p != "," {
						params = append(params, p)
					}
				}
			}
			*index++
			body := []string{}
			for *index < len(lines) && tokenize(lines[*index])[0] != "end" {
				body = append(body, lines[*index])
				*index++
			}
			functions[name] = Function{params, body}
		}

		// ===== Func Call =====
		if fn, ok := functions[cmd]; ok {
			args := []interface{}{}
			for _, arg := range tokens[1:] {
				args = append(args, evalExpr(arg))
			}
			backup := make(map[string]interface{})
			for k, v := range mutables {
				backup[k] = v
			}
			for i, p := range fn.params {
				mutables[p] = args[i]
			}
			idx := 0
			execBlock(fn.body, &idx)
			mutables = backup
		}

		// ===== Include =====
var includedFiles = map[string]bool{}

if cmd == "include" && len(tokens) >= 2 {
    filePath := strings.Trim(tokens[1], "\"")
    if includedFiles[filePath] {
        // Prevent re-including same file
        *index++
        continue
    }
    includedFiles[filePath] = true

    f, err := os.Open(filePath)
    if err != nil {
        fmt.Println("Include error:", err)
    } else {
        scanner := bufio.NewScanner(f)
        includeLines := []string{}
        for scanner.Scan() {
            includeLines = append(includeLines, scanner.Text())
        }
        f.Close()
        idx := 0
        execBlock(includeLines, &idx)
    }
}

		*index++
	}
}

func execute(lines []string) {
	index := 0
	execBlock(lines, &index)
}
func main() {
	program := []string{
		"print \"Hello from Falcon\"",
	}
	execute(program)
}
