
export interface ProblemDetails {
  status?: number;
  detail?: string;
  errors?: Array<{ message?: string }>;
}

export class ApiError extends Error {
  status: number;
  details: ProblemDetails | null;

  constructor(message: string, status: number, details: ProblemDetails | null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.details = details;
  }
}
