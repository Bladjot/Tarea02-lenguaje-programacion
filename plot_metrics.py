#!/usr/bin/env python3
"""
Genera visualizaciones a partir del CSV de métricas producido por main.go.

Ejemplo de uso:

    python plot_metrics.py metricas_espec.csv comparacion_estrategias.png
"""

from __future__ import annotations

import argparse
import csv
import statistics
from collections import defaultdict
from pathlib import Path
from typing import Dict, List

import matplotlib.pyplot as plt


VALID_MODES = {"especulativo", "secuencial"}


def load_totals(path: Path) -> Dict[str, List[float]]:
    """Carga los tiempos totales por modo desde el CSV."""
    per_mode: Dict[str, Dict[int, float]] = defaultdict(dict)

    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            mode = row.get("mode", "").strip().lower()
            if mode not in VALID_MODES:
                # Evita filas de resumen o vacías.
                continue
            run_str = row.get("run", "").strip()
            try:
                run_index = int(run_str)
            except ValueError:
                continue
            value = row.get("total_duration_ms", "").strip()
            if not value:
                continue
            try:
                per_mode[mode].setdefault(run_index, float(value))
            except ValueError:
                continue

    durations: Dict[str, List[float]] = {}
    for mode, runs in per_mode.items():
        durations[mode] = [runs[idx] for idx in sorted(runs.keys())]
    return durations


def build_figure(data: Dict[str, List[float]], title: str) -> plt.Figure:
    """Devuelve una figura con barras de promedios y un gráfico de líneas por corrida."""
    figure, axes = plt.subplots(1, 2, figsize=(10, 4))

    # Subgráfico de barras con promedios.
    ax_bar = axes[0]
    labels = []
    averages = []
    for mode in sorted(data.keys()):
        labels.append(mode.capitalize())
        averages.append(statistics.mean(data[mode]))
    ax_bar.bar(labels, averages, color=["steelblue", "darkorange"])
    ax_bar.set_ylabel("Tiempo promedio (ms)")
    ax_bar.set_title("Promedio por estrategia")

    # Subgráfico de líneas para observar la evolución por corrida.
    ax_line = axes[1]
    for mode in sorted(data.keys()):
        runs = list(range(1, len(data[mode]) + 1))
        ax_line.plot(runs, data[mode], marker="o", label=mode.capitalize())
    ax_line.set_xlabel("Corrida")
    ax_line.set_ylabel("Tiempo total (ms)")
    ax_line.set_title("Evolución por corrida")
    ax_line.legend()

    figure.suptitle(title)
    figure.tight_layout()
    return figure


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Genera gráficas comparativas a partir del CSV de métricas."
    )
    parser.add_argument("csv_path", type=Path, help="Ruta al archivo CSV generado por el programa.")
    parser.add_argument(
        "output",
        type=Path,
        nargs="?",
        default=Path("comparacion_estrategias.png"),
        help="Ruta del archivo de imagen a generar (PNG).",
    )
    parser.add_argument(
        "--title",
        default="Comparación ejecución especulativa vs secuencial",
        help="Título principal de la figura.",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    durations = load_totals(args.csv_path)
    if not durations:
        raise SystemExit("No se encontraron datos válidos en el CSV.")

    figure = build_figure(durations, title=args.title)
    figure.savefig(args.output, dpi=150, bbox_inches="tight")
    print(f"Gráfica generada en: {args.output}")


if __name__ == "__main__":
    main()
