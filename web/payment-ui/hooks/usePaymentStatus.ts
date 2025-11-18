"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Payment, PaymentEvent } from "@/types/payment";
import { fetchPaymentStatus } from "@/lib/api";
import { PaymentWebSocket } from "@/lib/websocket";

export function usePaymentStatus(paymentId: string) {
  const [payment, setPayment] = useState<Payment | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const router = useRouter();

  const loadPayment = useCallback(async () => {
    try {
      setIsLoading(true);
      const data = await fetchPaymentStatus(paymentId);
      setPayment(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setIsLoading(false);
    }
  }, [paymentId]);

  useEffect(() => {
    loadPayment();

    // Connect to WebSocket for real-time updates
    const ws = new PaymentWebSocket(paymentId);

    ws.on("payment.pending", (event: PaymentEvent) => {
      console.log("Payment pending:", event);
      setPayment((prev) =>
        prev
          ? {
              ...prev,
              status: "pending",
              tx_hash: event.tx_hash,
            }
          : null
      );
    });

    ws.on("payment.confirming", (event: PaymentEvent) => {
      console.log("Payment confirming:", event);
      setPayment((prev) =>
        prev
          ? {
              ...prev,
              status: "confirming",
              tx_hash: event.tx_hash,
              confirmations: event.confirmations || 0,
            }
          : null
      );
    });

    ws.on("payment.completed", (event: PaymentEvent) => {
      console.log("Payment completed:", event);
      setPayment((prev) =>
        prev
          ? {
              ...prev,
              status: "completed",
              tx_hash: event.tx_hash,
              confirmations: event.confirmations || 0,
            }
          : null
      );

      // Redirect to success page after a short delay
      setTimeout(() => {
        router.push(`/order/${paymentId}/success`);
      }, 2000);
    });

    ws.on("payment.expired", (event: PaymentEvent) => {
      console.log("Payment expired:", event);
      setPayment((prev) => (prev ? { ...prev, status: "expired" } : null));
    });

    ws.on("payment.failed", (event: PaymentEvent) => {
      console.log("Payment failed:", event);
      setPayment((prev) => (prev ? { ...prev, status: "failed" } : null));
    });

    return () => {
      ws.close();
    };
  }, [paymentId, loadPayment, router]);

  return { payment, isLoading, error };
}
