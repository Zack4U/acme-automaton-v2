package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/emicklei/dot"
	"github.com/sqweek/dialog"
)

type Automaton struct {
	States       []string                       `json:"states"`
	Alphabet     []string                       `json:"alphabet"`
	Transitions  map[string]map[string][]string `json:"transitions"`
	InitialState string                         `json:"initial"`
	FinalStates  []string                       `json:"final"`
}

var automaton Automaton

var dfa Automaton

func (a *Automaton) acceptsAFD(input string) bool {
	// Verificar automata original
	currentState := a.InitialState
	for _, symbol := range input {
		nextState, ok := a.Transitions[currentState][string(symbol)]
		if !ok {
			return false
		}
		currentState = nextState[0]
	}
	for _, finalState := range a.FinalStates {
		if currentState == finalState {
			return true
		}
	}
	return false
}

func (a *Automaton) acceptsAFND(input string) bool {
	return acceptStringRecursive(a, a.InitialState, input)
}

func acceptStringRecursive(a *Automaton, currentState string, input string) bool {
	if len(input) == 0 {
		return isFinalState(a, currentState)
	}

	nextSymbol := string(input[0])
	remainingInput := input[1:]

	transitions := a.Transitions[currentState]

	if nextState, exists := transitions[nextSymbol]; exists {
		for _, state := range nextState {
			if acceptStringRecursive(a, state, remainingInput) {
				return true
			}
		}
	}

	if nextState, exists := transitions["lda"]; exists {
		for _, state := range nextState {
			if acceptStringRecursive(a, state, input) {
				return true
			}
		}
	}

	return false
}

func isFinalState(a *Automaton, state string) bool {
	for _, finalState := range a.FinalStates {
		if finalState == state {
			return true
		}
	}
	return false
}


// Muestra un dialogo para seleccion de archivos
func showDialog(ext []string) (string, error) {
	// Obtiene el directorio actual
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Abre el dialogo de selección de archivo
	fileDialog := dialog.File().Title("Seleccionar archivo").Filter(ext[0], ext[1])
	filePath, err := fileDialog.SetStartDir(dir).Load()

	if err != nil {
		return "", err
	}

	// Retorna el path absoluto del archivo seleccionado
	return filepath.Abs(filePath)
}

// Permite cargar el archivo del automata
func upload() error {
	// Abre el dialogo de selección de archivo
	ext := []string{".json", "json"}

	if len(automaton.States) > 0 {
		automaton = Automaton{}
	}

	filePath, err := showDialog(ext)
	if err != nil {
		return errors.New("Error al seleccionar archivo")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return errors.New("Error al leer archivo JSON")
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.New("Error al leer archivo JSON")
	}

	err = json.Unmarshal(bytes, &automaton)
	if err != nil {
		return errors.New("Error al leer archivo JSON")
	}
	return nil
}

// Permite cargar el archivo de la cadena
func uploadString() (string, error) {
	
	// Abre el dialogo de selección de archivo
	ext := []string{".txt", "txt"}
	var cadena string
	filePath, err := showDialog(ext)
	if err != nil {
		return cadena, errors.New("Error al seleccionar archivo")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return cadena, errors.New("Error al leer archivo TXT")
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return cadena, errors.New("Error al leer archivo TXT")
	}

	cadena = string(bytes[:])
	cadena = strings.ToLower(cadena)
	return cadena, nil
}

// Convierte el automata en un String
func toString() string {
	automatonS, _ := json.MarshalIndent(automaton, "", " ")
	return string(automatonS)
}

// Crea el grafo con los valores del automata
func createGraph(a Automaton) (*dot.Graph, error) {
	g := dot.NewGraph(dot.Directed)

	// Generar los nodos
	for _, state := range a.States {
		node := g.Node(state)
		if state == a.InitialState {
			node.Attr("style", "filled")
			node.Attr("fillcolor", "green")
		}
		for _, finalState := range a.FinalStates {
			if state == finalState {
				node.Attr("style", "filled")
				node.Attr("fillcolor", "red")
			}
		}
	}

	// Generar las aristas
	for source, transitions := range a.Transitions {
		for symbol, targets := range transitions {
			for _, target := range targets {
				g.Edge(g.Node(source), g.Node(target)).Attr("label", symbol)
			}
		}
	}

	return g, nil
}

func ConvertToDFA(afn Automaton) Automaton {
	dfa = Automaton{
		States:       []string{},
		Alphabet:     afn.Alphabet,
		Transitions:  map[string]map[string][]string{},
		InitialState: "",
		FinalStates:  []string{},
	}

	// Función auxiliar para obtener el cierre-épsilon de un conjunto de estados
	epsilonClosure := func(states []string) []string {
		visited := map[string]bool{}
		var queue []string
		for _, state := range states {
			queue = append(queue, state)
			visited[state] = true
		}

		for len(queue) > 0 {
			currentState := queue[0]
			queue = queue[1:]

			transitions, ok := afn.Transitions[currentState]["lda"]
			if ok {
				for _, nextState := range transitions {
					if !visited[nextState] {
						queue = append(queue, nextState)
						visited[nextState] = true
					}
				}
			}
		}

		closure := []string{}
		for state := range visited {
			closure = append(closure, state)
		}
		return closure
	}

	// Función auxiliar para obtener las transiciones desde un conjunto de estados dado un símbolo de entrada
	getTransitions := func(states []string, symbol string) []string {
		transitions := []string{}
		for _, state := range states {
			if nextStates, ok := afn.Transitions[state][symbol]; ok {
				transitions = append(transitions, nextStates...)
			}
		}
		return transitions
	}

	// Obtener el conjunto inicial de estados
	initialStates := epsilonClosure([]string{afn.InitialState})

	// Crear el estado inicial del AFD
	initialStateName := strings.Join(initialStates, ",")
	dfa.States = append(dfa.States, initialStateName)
	dfa.InitialState = initialStateName

	// Mapa auxiliar para almacenar los conjuntos de estados del AFD y su correspondencia con el nombre del estado
	stateMap := map[string][]string{
		initialStateName: initialStates,
	}

	// Cola para el procesamiento de nuevos conjuntos de estados
	queue := []string{initialStateName}

	for len(queue) > 0 {
		currentStateName := queue[0]
		queue = queue[1:]

		currentState := stateMap[currentStateName]

		// Comprobar si el conjunto de estados actual contiene estados finales del AFN
		for _, finalState := range afn.FinalStates {
			if contains(currentState, finalState) && !contains(dfa.FinalStates, currentStateName) {
				dfa.FinalStates = append(dfa.FinalStates, currentStateName)
			}
		}

		// Procesar transiciones para cada símbolo del alfabeto
		for _, symbol := range afn.Alphabet {
			// Obtener el conjunto de estados alcanzables desde el estado actual con el símbolo dado
			nextStates := epsilonClosure(getTransitions(currentState, symbol))

			if len(nextStates) == 0 {
				continue
			}

			// Verificar si el conjunto de estados ya ha sido procesado
			nextStateName := strings.Join(nextStates, ",")
			if _, exists := stateMap[nextStateName]; !exists {
				// Agregar el nuevo conjunto de estados al AFD
				dfa.States = append(dfa.States, nextStateName)
				stateMap[nextStateName] = nextStates
				queue = append(queue, nextStateName)
			}

			// Agregar la transición desde el estado actual hacia el nuevo conjunto de estados
			if dfa.Transitions[currentStateName] == nil {
				dfa.Transitions[currentStateName] = map[string][]string{}
			}
			dfa.Transitions[currentStateName][symbol] = []string{nextStateName}
		}
	}

	return dfa
}

// Función auxiliar para verificar si un elemento se encuentra en un slice
func contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}


// Funcion principal
func main() {

	var app = app.New()

	window := app.NewWindow("Automatons by ACME")

	icon, err := fyne.LoadResourceFromPath("./Resources/upload.png")
	window.SetIcon(icon)

	window.Resize(fyne.NewSize(400, 400))

	startIcon, err := fyne.LoadResourceFromPath("./Resources/start.png")
    if err != nil {
        fmt.Println("Error al leer archivo SVG:", err)
        return
    }

	importIcon, err := fyne.LoadResourceFromPath("./Resources/upload.png")
	if err != nil {
		fmt.Println("Error al leer archivo SVG:", err)
	}

	importTxt := widget.NewLabel("Sube el automata en formato JSON")

	showIcon, err := fyne.LoadResourceFromPath("./Resources/show.png")
	if err != nil {
		fmt.Println("Error al leer archivo SVG:", err)
		return
	}

	convertIcon, err := fyne.LoadResourceFromPath("./Resources/convert.png")
	if err != nil {
		fmt.Println("Error al leer archivo SVG:", err)
		return
	}

	entryTxt := widget.NewLabel("Ingresa la cadena a verificar para el automata original: ")
    entry := widget.NewEntry()

	entryTxt2 := widget.NewLabel("Ingresa la cadena a verificar para el automata convertido: ")
    entry2 := widget.NewEntry()

	automatonTxt1 := widget.NewLabel("")
	automatonTxt2 := widget.NewLabel("")

	importStringBtn := widget.NewButtonWithIcon("CARGAR CADENA", importIcon, func() {
        cadena, err := uploadString()
        if err != nil{
            entry.SetText("ERROR al cargar el archivo")
        }
        entry.SetText(cadena)
    })

	importStringBtn2 := widget.NewButtonWithIcon("CARGAR CADENA 2", importIcon, func() {
        cadena, err := uploadString()
        if err != nil{
            entry2.SetText("ERROR al cargar el archivo")
        }
        entry2.SetText(cadena)
    })
    
    originalGraphImage := canvas.NewImageFromFile("grafo.png")
    originalGraphImage.FillMode = canvas.ImageFillContain
    originalGraphImage.SetMinSize(fyne.NewSize(500, 500))

    graphImage := canvas.NewImageFromFile("grafoC.png")
    graphImage.FillMode = canvas.ImageFillContain
    graphImage.SetMinSize(fyne.NewSize(500, 500))

	importBtn := widget.NewButtonWithIcon("CARGAR", importIcon, func() {
		if upload() != nil {
			entry.SetText("ERROR al cargar el archivo")
		}
	})

	show := widget.NewButtonWithIcon("VER AUTOMATA", showIcon, func() {
        g, err := createGraph(automaton)
        if err != nil {
            panic(err)
        }
    
        // Eliminar el archivo grafo.png si existe
        if _, err := os.Stat("grafo.png"); err == nil {
            if err := os.Remove("grafo.png"); err != nil {
                panic(err)
            }
        }
    
        // Generar una imagen PNG del grafo utilizando Graphviz
        cmd := exec.Command("dot", "-Tpng")
        var out bytes.Buffer
        cmd.Stdin = io.TeeReader(bytes.NewBufferString(g.String()), &out)
        outfile, err := os.Create("grafo.png")
        if err != nil {
            panic(err)
        }
        defer outfile.Close()
        cmd.Stdout = outfile
        if err := cmd.Run(); err != nil {
            panic(err)
        }
    
        originalGraphImage.File = "grafo.png"
        originalGraphImage.Refresh()
    })

    convert := widget.NewButtonWithIcon("CONVERTIR AUTOMATA", convertIcon, func() {
        automatonC := ConvertToDFA(automaton)
        g, err := createGraph(automatonC)
        if err != nil {
            panic(err)
        }
    
        // Eliminar el archivo grafo.png si existe
        if _, err := os.Stat("grafo.png"); err == nil {
            if err := os.Remove("grafo.png"); err != nil {
                panic(err)
            }
        }
    
        // Generar una imagen PNG del grafo utilizando Graphviz
        cmd := exec.Command("dot", "-Tpng")
        var out bytes.Buffer
        cmd.Stdin = io.TeeReader(bytes.NewBufferString(g.String()), &out)
        outfile, err := os.Create("grafoC.png")
        if err != nil {
            panic(err)
        }
        defer outfile.Close()
        cmd.Stdout = outfile
        if err := cmd.Run(); err != nil {
            panic(err)
        }
    
        graphImage.File = "grafoC.png"
        graphImage.Refresh()
    })

	start := widget.NewButtonWithIcon("VERIFICAR CADENAS", startIcon, func() {
        cadena1 := strings.ToLower(entry.Text)
		if cadena1 != "" {
			status := automaton.acceptsAFND(cadena1)
			if status {
				txt := fmt.Sprintf(">> LA CADENA ES VALIDA <<")
				automatonTxt1.SetText(txt)
				
			} else {
				txt := fmt.Sprintf(">> LA CADENA NO ES VALIDA <<")
				automatonTxt1.SetText(txt)
			}
			
		}
        cadena2 := strings.ToLower(entry2.Text)
		if cadena2 != "" {
			status := dfa.acceptsAFD(cadena2)
			if status {
				txt := fmt.Sprintf(">> LA CADENA ES VALIDA <<")
				automatonTxt2.SetText(txt)
				
			} else {
				txt := fmt.Sprintf(">> LA CADENA NO ES VALIDA <<", )
				automatonTxt2.SetText(txt)
			}
			
		}
    })

    middleTitle := widget.NewLabel("AUTOMATA ORIGINAL")
    rightTitle := widget.NewLabel("AUTOMATA CONVERTIDO")

	leftContent := container.NewVBox(importTxt, importBtn, entryTxt, importStringBtn, entry,automatonTxt1, entryTxt2, importStringBtn2, entry2, automatonTxt2, start, show, convert,)
    middleContent := container.NewVBox(middleTitle, originalGraphImage)
    rightContent := container.NewVBox(rightTitle, graphImage)

    contentContainer := container.NewHBox(leftContent, middleContent, rightContent)

	window.SetContent(contentContainer)
	window.ShowAndRun()

	defer func() {
		err := deleteFile("grafo.png")
		if err != nil {
			fmt.Println("Error eliminando el archivo grafo.png:", err)
		}
		err = deleteFile("grafoC.png")
		if err != nil {
			fmt.Println("Error eliminando el archivo grafoC.png:", err)
		}
	}()
}

func deleteFile(filename string) error {
	err := os.Remove(filename)
	if err != nil {
		return fmt.Errorf("error eliminando el archivo %s: %w", filename, err)
	}
	return nil
}


