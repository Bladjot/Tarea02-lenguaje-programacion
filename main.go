package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	branchA = "A"
	branchB = "B"
)

var (
	// ErrCancelled indica que la operación fue cancelada por el controlador principal.
	ErrCancelled = errors.New("branch cancelled")
)

// Config reúne los parámetros controlables desde la línea de comandos.
type Config struct {
	MatrixSize    int
	Threshold     int64
	OutputFile    string
	Runs          int
	PowDifficulty int
	PowData       string
	PrimesLimit   int
}

// BranchOutput encapsula la información relevante producida por un trabajo.
type BranchOutput struct {
	Numeric int64
	Detail  string
}

// BranchWork representa una carga de trabajo que puede reaccionar ante cancelaciones.
type BranchWork func(cancel <-chan struct{}) (BranchOutput, error)

// BranchResult almacena las métricas capturadas durante la ejecución de una rama.
type BranchResult struct {
	Name      string
	Numeric   int64
	Detail    string
	Start     time.Time
	End       time.Time
	Duration  time.Duration
	Cancelled bool
	Err       error
}

// ExecutionRun agrega la información relevante de una simulación completa (una corrida).
type ExecutionRun struct {
	Mode              string
	RunIndex          int
	ConditionValue    int64
	ConditionDuration time.Duration
	Winner            string
	TotalDuration     time.Duration
	RunStart          time.Time
	Branches          []BranchResult
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	branchWorks := buildBranchWorkload(cfg)

	specRuns := make([]ExecutionRun, 0, cfg.Runs)
	for i := 1; i <= cfg.Runs; i++ {
		run, err := runSpeculative(cfg, i, branchWorks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "speculative run %d failed: %v\n", i, err)
			os.Exit(1)
		}
		specRuns = append(specRuns, run)
	}

	seqRuns := make([]ExecutionRun, 0, cfg.Runs)
	for i := 1; i <= cfg.Runs; i++ {
		run, err := runSequential(cfg, i, branchWorks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sequential run %d failed: %v\n", i, err)
			os.Exit(1)
		}
		seqRuns = append(seqRuns, run)
	}

	if err := writeMetrics(cfg.OutputFile, specRuns, seqRuns); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing metrics: %v\n", err)
		os.Exit(1)
	}

	avgSpec := averageDuration(specRuns)
	avgSeq := averageDuration(seqRuns)
	speedup := computeSpeedup(avgSeq, avgSpec)

	fmt.Printf("Simulaciones completadas: %d (especulativo) + %d (secuencial)\n", len(specRuns), len(seqRuns))
	fmt.Printf("Promedio especulativo: %s\n", formatDuration(avgSpec))
	fmt.Printf("Promedio secuencial: %s\n", formatDuration(avgSeq))
	fmt.Printf("Speedup estimado: %.3f\n", speedup)
	fmt.Printf("Métricas almacenadas en: %s\n", cfg.OutputFile)
}

func parseFlags() Config {
	matrixSize := flag.Int("n", 125, "dimensión de las matrices cuadradas para la traza del producto")
	threshold := flag.Int64("umbral", 500000, "umbral para seleccionar la rama ganadora")
	output := flag.String("nombre_archivo", "metricas.csv", "archivo de salida para registrar las métricas")
	runs := flag.Int("runs", 30, "número de ejecuciones por estrategia")
	difficulty := flag.Int("difficulty", 5, "dificultad utilizada en la simulación de Proof-of-Work")
	data := flag.String("pow-data", "speculative", "dato base para el Proof-of-Work")
	primesLimit := flag.Int("primes-limit", 500000, "valor máximo para la búsqueda de números primos")
	flag.Parse()

	return Config{
		MatrixSize:    *matrixSize,
		Threshold:     *threshold,
		OutputFile:    *output,
		Runs:          *runs,
		PowDifficulty: *difficulty,
		PowData:       *data,
		PrimesLimit:   *primesLimit,
	}
}

func validateConfig(cfg Config) error {
	switch {
	case cfg.MatrixSize <= 0:
		return errors.New("n debe ser mayor que cero")
	case cfg.Runs <= 0:
		return errors.New("runs debe ser mayor que cero")
	case cfg.PowDifficulty <= 0:
		return errors.New("difficulty debe ser mayor que cero")
	case cfg.PrimesLimit <= 0:
		return errors.New("primes-limit debe ser mayor que cero")
	case strings.TrimSpace(cfg.OutputFile) == "":
		return errors.New("nombre_archivo no puede estar vacío")
	default:
		return nil
	}
}

func buildBranchWorkload(cfg Config) map[string]BranchWork {
	return map[string]BranchWork{
		branchA: func(cancel <-chan struct{}) (BranchOutput, error) {
			hash, nonce, err := SimularProofOfWorkWithCancel(cancel, cfg.PowData, cfg.PowDifficulty)
			if err != nil && !errors.Is(err, ErrCancelled) {
				return BranchOutput{}, err
			}
			detail := fmt.Sprintf("hash=%s", hash)
			return BranchOutput{
				Numeric: int64(nonce),
				Detail:  detail,
			}, err
		},
		branchB: func(cancel <-chan struct{}) (BranchOutput, error) {
			primes, err := EncontrarPrimosWithCancel(cancel, cfg.PrimesLimit)
			if err != nil && !errors.Is(err, ErrCancelled) {
				return BranchOutput{}, err
			}
			var detail string
			if len(primes) > 0 {
				detail = fmt.Sprintf("count=%d,last=%d", len(primes), primes[len(primes)-1])
			} else {
				detail = "count=0"
			}
			return BranchOutput{
				Numeric: int64(len(primes)),
				Detail:  detail,
			}, err
		},
	}
}

func runSpeculative(cfg Config, runIndex int, works map[string]BranchWork) (ExecutionRun, error) {
	workA, okA := works[branchA]
	workB, okB := works[branchB]
	if !okA || !okB {
		return ExecutionRun{}, errors.New("las dos ramas A y B deben estar definidas")
	}

	runStart := time.Now()
	resultsCh := make(chan BranchResult, 2)

	cancelA := make(chan struct{})
	cancelB := make(chan struct{})

	go executeBranchAsync(branchA, workA, cancelA, resultsCh)
	go executeBranchAsync(branchB, workB, cancelB, resultsCh)

	conditionStart := time.Now()
	trace := int64(CalcularTrazaDeProductoDeMatrices(cfg.MatrixSize))
	conditionDuration := time.Since(conditionStart)

	winner := chooseBranch(trace, cfg.Threshold)
	if winner == branchA {
		close(cancelB)
	} else {
		close(cancelA)
	}

	var branches []BranchResult
	for len(branches) < 2 {
		result := <-resultsCh
		if result.Err != nil {
			return ExecutionRun{}, fmt.Errorf("branch %s failed: %w", result.Name, result.Err)
		}
		branches = append(branches, result)
	}

	totalDuration := time.Since(runStart)

	return ExecutionRun{
		Mode:              "especulativo",
		RunIndex:          runIndex,
		ConditionValue:    trace,
		ConditionDuration: conditionDuration,
		Winner:            winner,
		TotalDuration:     totalDuration,
		RunStart:          runStart,
		Branches:          branches,
	}, nil
}

func runSequential(cfg Config, runIndex int, works map[string]BranchWork) (ExecutionRun, error) {
	runStart := time.Now()

	conditionStart := time.Now()
	trace := int64(CalcularTrazaDeProductoDeMatrices(cfg.MatrixSize))
	conditionDuration := time.Since(conditionStart)

	winner := chooseBranch(trace, cfg.Threshold)
	work, ok := works[winner]
	if !ok {
		return ExecutionRun{}, fmt.Errorf("no existe la rama %s", winner)
	}

	result := executeBranchSync(winner, work)
	if result.Err != nil {
		return ExecutionRun{}, fmt.Errorf("branch %s failed: %w", result.Name, result.Err)
	}

	totalDuration := time.Since(runStart)

	return ExecutionRun{
		Mode:              "secuencial",
		RunIndex:          runIndex,
		ConditionValue:    trace,
		ConditionDuration: conditionDuration,
		Winner:            winner,
		TotalDuration:     totalDuration,
		RunStart:          runStart,
		Branches:          []BranchResult{result},
	}, nil
}

func executeBranchAsync(name string, work BranchWork, cancel <-chan struct{}, out chan<- BranchResult) {
	start := time.Now()
	output, err := work(cancel)
	end := time.Now()

	result := BranchResult{
		Name:     name,
		Numeric:  output.Numeric,
		Detail:   output.Detail,
		Start:    start,
		End:      end,
		Duration: end.Sub(start),
	}

	switch {
	case errors.Is(err, ErrCancelled):
		result.Cancelled = true
	case err != nil:
		result.Err = err
	}

	out <- result
}

func executeBranchSync(name string, work BranchWork) BranchResult {
	start := time.Now()
	output, err := work(nil)
	end := time.Now()

	result := BranchResult{
		Name:     name,
		Numeric:  output.Numeric,
		Detail:   output.Detail,
		Start:    start,
		End:      end,
		Duration: end.Sub(start),
	}

	switch {
	case errors.Is(err, ErrCancelled):
		result.Cancelled = true
	case err != nil:
		result.Err = err
	}

	return result
}

func writeMetrics(path string, specRuns, seqRuns []ExecutionRun) error {
	if err := os.MkdirAll(directory(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"mode",
		"run",
		"branch",
		"was_winner",
		"cancelled",
		"result_numeric",
		"result_detail",
		"condition_value",
		"condition_duration_ms",
		"branch_start_ms",
		"branch_end_ms",
		"branch_duration_ms",
		"total_duration_ms",
		"error",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	writeRun := func(run ExecutionRun) error {
		for _, branch := range run.Branches {
			startOffset := branch.Start.Sub(run.RunStart).Seconds() * 1000
			endOffset := branch.End.Sub(run.RunStart).Seconds() * 1000
			record := []string{
				run.Mode,
				strconv.Itoa(run.RunIndex),
				branch.Name,
				boolToString(branch.Name == run.Winner),
				boolToString(branch.Cancelled),
				strconv.FormatInt(branch.Numeric, 10),
				branch.Detail,
				strconv.FormatInt(run.ConditionValue, 10),
				floatToString(run.ConditionDuration.Seconds() * 1000),
				floatToString(startOffset),
				floatToString(endOffset),
				floatToString(branch.Duration.Seconds() * 1000),
				floatToString(run.TotalDuration.Seconds() * 1000),
				errorString(branch.Err),
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
		return nil
	}

	for _, run := range specRuns {
		if err := writeRun(run); err != nil {
			return err
		}
	}
	for _, run := range seqRuns {
		if err := writeRun(run); err != nil {
			return err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	avgSpec := averageDuration(specRuns)
	avgSeq := averageDuration(seqRuns)
	speedup := computeSpeedup(avgSeq, avgSpec)

	if err := writer.Write([]string{}); err != nil {
		return err
	}
	summary := []string{
		"resumen",
		"",
		"",
		"",
		"",
		fmt.Sprintf("avg_numeric_speculative=%.3f", averageNumeric(specRuns)),
		fmt.Sprintf("avg_numeric_sequential=%.3f", averageNumeric(seqRuns)),
		"",
		"",
		"",
		"",
		"",
		fmt.Sprintf("avg_speculative_ms=%.3f;avg_sequential_ms=%.3f;speedup=%.3f",
			avgSpec.Seconds()*1000,
			avgSeq.Seconds()*1000,
			speedup),
		"",
	}
	if err := writer.Write(summary); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func chooseBranch(trace, threshold int64) string {
	if trace >= threshold {
		return branchA
	}
	return branchB
}

// SimularProofOfWork simula la búsqueda de un hash con prefijo de ceros, tal como se entrega en el anexo.
func SimularProofOfWork(blockData string, dificultad int) (string, int) {
	hash, nonce, _ := SimularProofOfWorkWithCancel(nil, blockData, dificultad)
	return hash, nonce
}

// SimularProofOfWorkWithCancel es una variante que permite cancelación cooperativa.
func SimularProofOfWorkWithCancel(cancel <-chan struct{}, blockData string, dificultad int) (string, int, error) {
	targetPrefix := strings.Repeat("0", dificultad)
	nonce := 0

	for {
		if cancel != nil {
			select {
			case <-cancel:
				return "", 0, ErrCancelled
			default:
			}
		}

		data := fmt.Sprintf("%s%d", blockData, nonce)
		hashBytes := sha256.Sum256([]byte(data))
		hashString := hex.EncodeToString(hashBytes[:])

		if strings.HasPrefix(hashString, targetPrefix) {
			return hashString, nonce, nil
		}
		nonce++

		if cancel != nil && nonce%1_000 == 0 {
			select {
			case <-cancel:
				return "", 0, ErrCancelled
			default:
			}
		}
	}
}

// EncontrarPrimos devuelve la lista de números primos hasta max, siguiendo el anexo.
func EncontrarPrimos(max int) []int {
	primes, _ := EncontrarPrimosWithCancel(nil, max)
	return primes
}

// EncontrarPrimosWithCancel es una variante que añade soporte para cancelación cooperativa.
func EncontrarPrimosWithCancel(cancel <-chan struct{}, max int) ([]int, error) {
	if max < 2 {
		return []int{}, nil
	}

	primes := make([]int, 0, max/10)
	for i := 2; i < max; i++ {
		if cancel != nil {
			select {
			case <-cancel:
				return nil, ErrCancelled
			default:
			}
		}

		isPrime := true
		upper := int(math.Sqrt(float64(i)))
		for j := 2; j <= upper; j++ {
			if cancel != nil && j%1024 == 0 {
				select {
				case <-cancel:
					return nil, ErrCancelled
				default:
				}
			}
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			primes = append(primes, i)
		}
	}
	return primes, nil
}

// CalcularTrazaDeProductoDeMatrices multiplica dos matrices NxN con valores aleatorios y devuelve la traza.
func CalcularTrazaDeProductoDeMatrices(n int) int {
	m1 := make([][]int, n)
	m2 := make([][]int, n)
	for i := 0; i < n; i++ {
		m1[i] = make([]int, n)
		m2[i] = make([]int, n)
		for j := 0; j < n; j++ {
			m1[i][j] = rand.Intn(10)
			m2[i][j] = rand.Intn(10)
		}
	}

	trace := 0
	for i := 0; i < n; i++ {
		sum := 0
		for k := 0; k < n; k++ {
			sum += m1[i][k] * m2[k][i]
		}
		trace += sum
	}
	return trace
}

func averageDuration(runs []ExecutionRun) time.Duration {
	if len(runs) == 0 {
		return 0
	}
	var total time.Duration
	for _, run := range runs {
		total += run.TotalDuration
	}
	return total / time.Duration(len(runs))
}

func averageNumeric(runs []ExecutionRun) float64 {
	if len(runs) == 0 {
		return 0
	}
	var total float64
	var count int
	for _, run := range runs {
		for _, branch := range run.Branches {
			total += float64(branch.Numeric)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func computeSpeedup(sequential, speculative time.Duration) float64 {
	if speculative <= 0 {
		return 0
	}
	return sequential.Seconds() / speculative.Seconds()
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.3f ms", d.Seconds()*1000)
}

func directory(path string) string {
	lastSep := strings.LastIndex(path, string(os.PathSeparator))
	if lastSep == -1 {
		return "."
	}
	return path[:lastSep]
}

func boolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func floatToString(value float64) string {
	return fmt.Sprintf("%.3f", value)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
