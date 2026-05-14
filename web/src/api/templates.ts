import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type {
  StrategyTemplateDetailDTO,
  StrategyTemplatesListDTO,
} from "@/types/strategies";

export async function fetchTemplates(): Promise<StrategyTemplatesListDTO> {
  return apiFetch<StrategyTemplatesListDTO>(API_PATHS.console.templates);
}

export async function fetchTemplateDetail(
  id: string | number,
): Promise<StrategyTemplateDetailDTO> {
  return apiFetch<StrategyTemplateDetailDTO>(API_PATHS.console.template(id));
}
