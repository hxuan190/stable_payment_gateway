export type PaymentStatus =
  | "created"
  | "pending"
  | "confirming"
  | "completed"
  | "expired"
  | "failed";

export interface Payment {
  id: string;
  status: PaymentStatus;
  amount_crypto: string;
  amount_vnd: string;
  currency: string; // "USDT", "USDC"
  chain: string; // "solana", "bsc"
  wallet_address: string;
  payment_memo: string;
  qr_code_data: string;
  tx_hash?: string;
  confirmations: number;
  expires_at: string;
  created_at: string;
}

export interface PaymentEvent {
  type: string;
  payment_id: string;
  status: PaymentStatus;
  tx_hash?: string;
  confirmations?: number;
  timestamp: string;
}

export interface APIResponse<T> {
  data: T | null;
  error: {
    code: string;
    message: string;
  } | null;
  timestamp: string;
}
