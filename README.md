# Tarea 02 – Ejecución Especulativa con Go

## Descripción
Este proyecto implementa el patrón de **ejecución especulativa** propuesto en la consigna. El programa lanza dos goroutines (`rama_A` y `rama_B`) que resuelven tareas de cómputo intensivo en paralelo, mientras el hilo principal evalúa una condición costosa. Una vez que se conoce el resultado de la condición, la rama perdedora es cancelada mediante canales y se conserva únicamente el resultado de la rama ganadora. Además, se implementa una versión secuencial para comparar tiempos y calcular el *speedup*.

Las funciones de trabajo corresponden a las provistas en el anexo, adaptadas para soportar cancelación cooperativa:

- `SimularProofOfWork` / `SimularProofOfWorkWithCancel`: búsqueda de *nonce* con SHA-256 y prefijo de ceros.
- `EncontrarPrimos` / `EncontrarPrimosWithCancel`: conteo de números primos mediante división sucesiva.
- `CalcularTrazaDeProductoDeMatrices`: multiplicación de matrices aleatorias de tamaño `n × n` para calcular la traza.

## Requisitos
- Go ≥ 1.21
- (Opcional) Python ≥ 3.9 y `matplotlib` para graficar los resultados.

## Uso
```bash
go run . \
  -n 125 \
  -umbral 500000 \
  -nombre_archivo metricas.csv \
  -runs 30 \
  -difficulty 5 \
  -pow-data speculative \
  -primes-limit 500000
```

### Flags relevantes
- `-n`: dimensión de las matrices para `CalcularTrazaDeProductoDeMatrices`.
- `-umbral`: se compara contra la traza para decidir la rama ganadora (`>=` elige la rama A).
- `-nombre_archivo`: archivo CSV donde se guardan todas las métricas.
- `-runs`: número de corridas por estrategia (especulativa/secuencial).
- `-difficulty`: número de ceros iniciales en el hash del Proof-of-Work.
- `-pow-data`: dato base concatenado en el Proof-of-Work.
- `-primes-limit`: cota superior para la búsqueda de primos.

El programa imprime en consola el promedio de cada estrategia y el speedup estimado; la información detallada queda en el CSV indicado.

## Archivo de métricas
Cada fila del CSV representa el resultado de una rama:

| Campo | Descripción |
| --- | --- |
| `mode` | `especulativo` o `secuencial`. |
| `run` | Número de corrida (1…`runs`). |
| `branch` | Identificador (`A` o `B`). |
| `was_winner` | `true` si la rama fue la ganadora. |
| `cancelled` | `true` cuando la rama terminó por cancelación. |
| `result_numeric` | Valor numérico (nonce hallado o cantidad de primos). |
| `result_detail` | Texto con información adicional (hash encontrado, último primo, etc.). |
| `condition_value` | Valor de la traza usada para decidir la rama. |
| `condition_duration_ms` | Tiempo de la evaluación de la condición. |
| `branch_start_ms`, `branch_end_ms`, `branch_duration_ms` | Métricas temporales relativas al inicio de la corrida. |
| `total_duration_ms` | Duración total de la corrida (misma para todas las ramas reportadas). |
| `error` | Mensaje de error si correspondiera (en ejecuciones exitosas queda vacío). |

Al final del archivo se agrega una fila tipo `resumen` con los promedios y el speedup calculado automáticamente por el programa.

## Análisis de rendimiento
1. Ejecute el programa con el conjunto de parámetros que desee estudiar (ej. los valores por defecto).
2. Obtenga los promedios desde la fila `resumen` del CSV o desde la salida estándar.
3. Complete la siguiente tabla en su informe:

| Estrategia | Promedio total (ms) | Speedup |
| --- | --- | --- |
| Especulativa | 21.484 | 1.159 |
| Secuencial | 24.903 | 1.000 |

> Valores obtenidos con `go run . -n 400 -umbral 1 -runs 30 -difficulty 5 -pow-data casoA -primes-limit 500000` (archivo `metricas_caseA.csv`).

4. Analice también la variabilidad (desviación estándar) y discuta escenarios donde la especulación puede no aportar beneficios (por ejemplo, cuando las ramas tienen costos muy dispares).

## Gráficas sugeridas
Se incluye `plot_metrics.py`, que genera un archivo PNG con un gráfico de barras (promedios) y un gráfico de líneas (evolución por corrida) para los tiempos totales. Requiere Python y `matplotlib`.

```bash
python plot_metrics.py metricas.csv comparacion_estrategias.png
```

La imagen `comparacion_estrategias.png` puede incorporarse al reporte final para visualizar el speedup observado.

## Enlace al repositorio
Actualice el siguiente enlace con la URL de su repositorio Git hospedado:

```
https://github.com/bladjot/Tarea02-lenguaje-programacion
```

---

