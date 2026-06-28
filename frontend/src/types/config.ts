// Per-lane harness/model configuration (effective merged value + env defaults).
export interface PhaseConfig {
  phase: string;
  model: string;
  harness_cmd: string;
  max_retries: number;
  timeout_sec: number;
  default_model: string;
  default_harness_cmd: string;
  default_max_retries: number;
  default_timeout_sec: number;
}

// Editable input for one lane (empty/0 = inherit env default).
export interface PhaseConfigInput {
  phase: string;
  model: string;
  harness_cmd: string;
  max_retries: number;
  timeout_sec: number;
}