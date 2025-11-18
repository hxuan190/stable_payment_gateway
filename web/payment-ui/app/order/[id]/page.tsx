"use client";

import { useParams } from "next/navigation";
import { usePaymentStatus } from "@/hooks/usePaymentStatus";
import { PaymentStatus } from "@/components/PaymentStatus";

export default function PaymentPage() {
  const params = useParams();
  const paymentId = params.id as string;

  const { payment, isLoading, error } = usePaymentStatus(paymentId);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading payment details...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-6 text-center">
          <div className="text-4xl mb-4">❌</div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            Payment Not Found
          </h1>
          <p className="text-gray-600 mb-4">
            {error.message || "The payment you're looking for doesn't exist."}
          </p>
          <p className="text-sm text-gray-500">Payment ID: {paymentId}</p>
        </div>
      </div>
    );
  }

  if (!payment) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600">No payment data available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8 px-4">
      <PaymentStatus payment={payment} />
    </div>
  );
}
