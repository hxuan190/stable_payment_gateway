import { Payment, APIResponse } from "@/types/payment";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export class APIError extends Error {
  constructor(
    public code: string,
    message: string
  ) {
    super(message);
    this.name = "APIError";
  }
}

export async function fetchPaymentStatus(
  paymentId: string
): Promise<Payment> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/payments/${paymentId}/status`,
    {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    }
  );

  if (!response.ok) {
    if (response.status === 404) {
      throw new APIError("NOT_FOUND", "Payment not found");
    }
    throw new APIError("FETCH_ERROR", "Failed to fetch payment status");
  }

  const result: APIResponse<Payment> = await response.json();

  if (result.error) {
    throw new APIError(result.error.code, result.error.message);
  }

  if (!result.data) {
    throw new APIError("NO_DATA", "No payment data received");
  }

  return result.data;
}
