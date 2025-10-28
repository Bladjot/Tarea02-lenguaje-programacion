# Tarea 02 – Ejecución Especulativa con Go

## Descripción
El siguiente proyecto implementa el patrón de **ejecución especulativa** propuesto en el enunciado de la Tarea 2. El programa creado ejecuta dos goroutines (`rama_A` y `rama_B`) que resuelven tareas de cómputo intenso en paralelo, mientras un hilo principal evalúa una condición costosa para determinar cual de las 2 rutinas es la ganadora. Una vez que se conoce cual rama es la ganadora, la rama perdedora es cancelada mediante canales y se conserva únicamente el resultado de la rama ganadora. Además, se implementa una versión secuencial del programa para comparar tiempos y calcular el *speedup*.

Las funciones de trabajo corresponden a las provistas en el anexo, adaptadas para soportar cancelación cooperativa:

- `SimularProofOfWork` / `SimularProofOfWorkWithCancel`: búsqueda de *nonce* con SHA-256 y prefijo de ceros.
- `EncontrarPrimos` / `EncontrarPrimosWithCancel`: conteo de números primos mediante división sucesiva.
- `CalcularTrazaDeProductoDeMatrices`: multiplicación de matrices aleatorias de tamaño `n × n` para calcular la traza.

## Requisitos del programa
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

### Flags importantes
- `-n`: Esta flag determina la dimensión de las matrices para `CalcularTrazaDeProductoDeMatrices`.
- `-umbral`: Esta flag se usa para compararla con la traza para decidir la rama ganadora (`>=` elige la rama A).
- `-nombre_archivo`: Esta flag guardará en un archivo CSV las métricas obtenidas.
- `-runs`: Esta flag introduce un número de corridas por estrategia (especulativa/secuencial).
- `-difficulty`: Esta flag introduce un número de ceros iniciales en el hash del Proof-of-Work.
- `-pow-data`: Esta flag es el dato base concatenado en el Proof-of-Work.
- `-primes-limit`: Esta flag es la cota superior para la búsqueda de primos.

Cuando el programa termina este imprime en consola el promedio de cada estrategia y el speedup estimado, la información obtenida queda en un archivo CSV.

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
Para analizar el rendimiento del programa se deben ejecutar los siguientes pasos:
1. Ejecute el programa con el conjunto de parámetros que desee estudiar (ej. los valores por defecto).
2. Obtenga los promedios desde la fila `resumen` del CSV o desde la salida estándar.
3. Complete la siguiente tabla en su informe:

| Estrategia | Promedio total (ms) | Speedup |
| --- | --- | --- |
| Especulativa | 21.484 | 1.159 |
| Secuencial | 24.903 | 1.000 |

> Valores obtenidos con `go run . -n 400 -umbral 1 -runs 30 -difficulty 5 -pow-data casoA -primes-limit 500000` (archivo `metricas_caseA.csv`).

## Gráficos
Se incluyo en esta tarea un archhivo que incluye `plot_metrics.py`, este genera un archivo PNG con un gráfico de barras (promedios) y un gráfico de líneas (evolución por corrida) para los tiempos totales. Para esto se requiere Python y `matplotlib`.

La ejecución del archivo `plot_metrics.py` es con el siguiente comando:
```
python plot_metrics.py metricas.csv comparacion_estrategias.png
```

## Enlace al repositorio
```
https://github.com/bladjot/Tarea02-lenguaje-programacion
```
