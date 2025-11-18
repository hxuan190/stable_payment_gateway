"use client";

import { Payment } from "@/types/payment";
import { QRCode } from "./QRCode";
import { CountdownTimer } from "./CountdownTimer";

interface PaymentStatusProps {
  payment: Payment;
}

const statusConfig = {
  created: {
    label: "Awaiting Payment",
    color: "bg-gray-100 text-gray-800",
    icon: "⏳",
  },
  pending: {
    label: "Transaction Detected",
    color: "bg-yellow-100 text-yellow-800",
    icon: "🔍",
  },
  confirming: {
    label: "Confirming Payment",
    color: "bg-blue-100 text-blue-800",
    icon: "⏳",
  },
  completed: {
    label: "Payment Completed",
    color: "bg-green-100 text-green-800",
    icon: "✅",
  },
  expired: {
    label: "Payment Expired",
    color: "bg-red-100 text-red-800",
    icon: "❌",
  },
  failed: {
    label: "Payment Failed",
    color: "bg-red-100 text-red-800",
    icon: "❌",
  },
};

export function PaymentStatus({ payment }: PaymentStatusProps) {
  const config = statusConfig[payment.status];

  const getBlockExplorerUrl = () => {
    if (!payment.tx_hash) return null;

    if (payment.chain === "solana") {
      return `https://solscan.io/tx/${payment.tx_hash}`;
    } else if (payment.chain === "bsc") {
      return `https://bscscan.com/tx/${payment.tx_hash}`;
    }
    return null;
  };

  return (
    <div className="max-w-2xl mx-auto">
      {/* Status Header */}
      <div className="bg-white rounded-lg shadow-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold mb-2">Payment Details</h1>
            <p className="text-sm text-gray-600">ID: {payment.id}</p>
          </div>
          <div
            className={`px-4 py-2 rounded-full ${config.color} font-semibold flex items-center gap-2`}
          >
            <span>{config.icon}</span>
            <span>{config.label}</span>
          </div>
        </div>

        {/* Amount Display */}
        <div className="border-t border-gray-200 pt-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-sm text-gray-600 mb-1">Amount to Pay</div>
              <div className="text-2xl font-bold">
                {payment.amount_crypto} {payment.currency}
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-600 mb-1">
                VND Equivalent
              </div>
              <div className="text-2xl font-bold">
                {parseFloat(payment.amount_vnd).toLocaleString("vi-VN")} VND
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Countdown Timer */}
      {payment.status === "created" && (
        <div className="mb-6">
          <CountdownTimer expiresAt={payment.expires_at} />
        </div>
      )}

      {/* QR Code - Only show if payment is not completed or expired */}
      {(payment.status === "created" || payment.status === "pending") && (
        <div className="mb-6">
          <QRCode
            data={payment.qr_code_data}
            walletAddress={payment.wallet_address}
            amount={payment.amount_crypto}
            memo={payment.payment_memo}
            currency={payment.currency}
            chain={payment.chain}
          />
        </div>
      )}

      {/* Transaction Details */}
      {payment.tx_hash && (
        <div className="bg-white rounded-lg shadow-lg p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">Transaction Details</h3>

          <div className="space-y-3">
            <div>
              <div className="text-sm text-gray-600 mb-1">Transaction Hash</div>
              <div className="font-mono text-sm break-all bg-gray-50 p-2 rounded">
                {payment.tx_hash}
              </div>
              {getBlockExplorerUrl() && (
                <a
                  href={getBlockExplorerUrl()!}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-800 text-sm mt-1 inline-block"
                >
                  View on Block Explorer →
                </a>
              )}
            </div>

            {payment.status === "confirming" && (
              <div>
                <div className="text-sm text-gray-600 mb-1">Confirmations</div>
                <div className="flex items-center gap-2">
                  <div className="flex-1 bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{
                        width: `${Math.min((payment.confirmations / 12) * 100, 100)}%`,
                      }}
                    />
                  </div>
                  <span className="text-sm font-semibold">
                    {payment.confirmations}/12
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Instructions */}
      {payment.status === "created" && (
        <div className="bg-white rounded-lg shadow-lg p-6">
          <h3 className="text-lg font-semibold mb-4">How to Pay</h3>
          <ol className="list-decimal list-inside space-y-2 text-sm text-gray-700">
            <li>Open your {payment.chain === "solana" ? "Solana" : "BSC"} wallet</li>
            <li>Scan the QR code or copy the wallet address</li>
            <li>Send exactly {payment.amount_crypto} {payment.currency}</li>
            <li>
              <strong>Important:</strong> Include the payment reference in the
              memo field
            </li>
            <li>Wait for confirmation (usually 10-30 seconds)</li>
          </ol>
        </div>
      )}

      {/* Completed Message */}
      {payment.status === "completed" && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-6 text-center">
          <div className="text-4xl mb-2">🎉</div>
          <h3 className="text-xl font-bold text-green-800 mb-2">
            Payment Successful!
          </h3>
          <p className="text-green-700">
            Redirecting to confirmation page...
          </p>
        </div>
      )}

      {/* Expired/Failed Message */}
      {(payment.status === "expired" || payment.status === "failed") && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <div className="text-4xl mb-2">😞</div>
          <h3 className="text-xl font-bold text-red-800 mb-2">
            {payment.status === "expired"
              ? "Payment Expired"
              : "Payment Failed"}
          </h3>
          <p className="text-red-700 mb-4">
            {payment.status === "expired"
              ? "This payment link has expired. Please create a new payment."
              : "This payment could not be completed. Please try again."}
          </p>
        </div>
      )}
    </div>
  );
}
