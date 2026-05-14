import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type {
  BacktestCreateResponseDTO,
  BacktestJobDetailResponseDTO,
  BacktestJobsListDTO,
  BacktestRequestDTO,
  BacktestRerunResponseDTO,
} from "@/types/backtests";

export async function fetchBacktestJobs(): Promise<BacktestJobsListDTO> {
  return apiFetch<BacktestJobsListDTO>(API_PATHS.console.backtests);
}

export async function fetchBacktestJobDetail(
  id: string | number,
): Promise<BacktestJobDetailResponseDTO> {
  return apiFetch<BacktestJobDetailResponseDTO>(
    API_PATHS.console.backtest(id),
  );
}

export async function createBacktestJob(
  body: BacktestRequestDTO,
): Promise<BacktestCreateResponseDTO> {
  return apiFetch<BacktestCreateResponseDTO>(API_PATHS.console.backtests, {
    method: "POST",
    json: body,
  });
}

export async function backtestJobAction(
  id: string | number,
  action: "pause" | "terminate" | "cancel" | "rerun",
): Promise<{ status: string } | BacktestRerunResponseDTO> {
  return apiFetch<{ status: string } | BacktestRerunResponseDTO>(
    API_PATHS.console.backtestActions(id),
    { method: "POST", json: { action } },
  );
}
