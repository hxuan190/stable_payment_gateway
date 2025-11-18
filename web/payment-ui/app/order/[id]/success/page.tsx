"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { Payment } from "@/types/payment";
import { fetchPaymentStatus } from "@/lib/api";

export default function SuccessPage() {
  const params = useParams();
  const paymentId = params.id as string;
  const [payment, setPayment] = useState<Payment | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadPayment = async () => {
      try {
        const data = await fetchPaymentStatus(paymentId);
        setPayment(data);
      } catch (error) {
        console.error("Failed to load payment:", error);
      } finally {
        setIsLoading(false);
      }
    };

    loadPayment();
  }, [paymentId]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-green-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading receipt...</p>
        </div>
      </div>
    );
  }

  if (!payment) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600">Receipt not available</p>
        </div>
      </div>
    );
  }

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
    <div className="min-h-screen bg-gray-50 py-8 px-4">
      <div className="max-w-2xl mx-auto">
        {/* Success Header */}
        <div className="bg-white rounded-lg shadow-lg p-8 mb-6 text-center">
          <div className="text-6xl mb-4">✅</div>
          <h1 className="text-3xl font-bold text-green-600 mb-2">
            Payment Completed!
          </h1>
          <p className="text-gray-600">
            Your payment has been successfully processed
          </p>
        </div>

        {/* Receipt Details */}
        <div className="bg-white rounded-lg shadow-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4 pb-2 border-b">
            Payment Receipt
          </h2>

          <div className="space-y-4">
            <div className="flex justify-between py-2 border-b">
              <span className="text-gray-600">Payment ID</span>
              <span className="font-mono text-sm">{payment.id}</span>
            </div>

            <div className="flex justify-between py-2 border-b">
              <span className="text-gray-600">Amount Paid</span>
              <div className="text-right">
                <div className="font-semibold">
                  {payment.amount_crypto} {payment.currency}
                </div>
                <div className="text-sm text-gray-500">
                  ≈ {parseFloat(payment.amount_vnd).toLocaleString("vi-VN")} VND
                </div>
              </div>
            </div>

            <div className="flex justify-between py-2 border-b">
              <span className="text-gray-600">Chain</span>
              <span className="font-semibold capitalize">{payment.chain}</span>
            </div>

            {payment.tx_hash && (
              <div className="py-2 border-b">
                <div className="text-gray-600 mb-2">Transaction Hash</div>
                <div className="font-mono text-xs break-all bg-gray-50 p-2 rounded mb-2">
                  {payment.tx_hash}
                </div>
                {getBlockExplorerUrl() && (
                  <a
                    href={getBlockExplorerUrl()!}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 text-sm inline-flex items-center gap-1"
                  >
                    <span>View on Block Explorer</span>
                    <svg
                      className="w-4 h-4"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                      />
                    </svg>
                  </a>
                )}
              </div>
            )}

            <div className="flex justify-between py-2 border-b">
              <span className="text-gray-600">Payment Date</span>
              <span className="font-semibold">
                {new Date(payment.created_at).toLocaleString("vi-VN")}
              </span>
            </div>

            <div className="flex justify-between py-2">
              <span className="text-gray-600">Status</span>
              <span className="px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm font-semibold">
                Completed
              </span>
            </div>
          </div>
        </div>

        {/* Important Notice */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 mb-6">
          <h3 className="text-lg font-semibold text-blue-900 mb-2">
            Important Information
          </h3>
          <ul className="text-sm text-blue-800 space-y-2">
            <li className="flex items-start gap-2">
              <span className="text-blue-600 mt-0.5">•</span>
              <span>
                Keep this receipt for your records. You can use the transaction
                hash to verify the payment on the blockchain.
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-blue-600 mt-0.5">•</span>
              <span>
                The merchant has been notified and will process your order
                shortly.
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-blue-600 mt-0.5">•</span>
              <span>
                If you have any questions about your purchase, please contact
                the merchant directly.
              </span>
            </li>
          </ul>
        </div>

        {/* Actions */}
        <div className="flex gap-4">
          <button
            onClick={() => window.print()}
            className="flex-1 bg-blue-600 text-white py-3 px-6 rounded-lg font-semibold hover:bg-blue-700 transition"
          >
            Print Receipt
          </button>
          <button
            onClick={() => (window.location.href = "/")}
            className="flex-1 bg-gray-200 text-gray-800 py-3 px-6 rounded-lg font-semibold hover:bg-gray-300 transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
