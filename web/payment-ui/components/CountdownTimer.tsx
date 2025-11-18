"use client";

import { useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";

interface CountdownTimerProps {
  expiresAt: string;
  onExpire?: () => void;
}

export function CountdownTimer({ expiresAt, onExpire }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState<string>("");
  const [isExpired, setIsExpired] = useState(false);

  useEffect(() => {
    const updateTimer = () => {
      const expiryTime = new Date(expiresAt).getTime();
      const now = new Date().getTime();
      const difference = expiryTime - now;

      if (difference <= 0) {
        setIsExpired(true);
        setTimeLeft("Expired");
        if (onExpire) {
          onExpire();
        }
        return;
      }

      const minutes = Math.floor((difference % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((difference % (1000 * 60)) / 1000);

      setTimeLeft(`${minutes}m ${seconds}s`);
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);

    return () => clearInterval(interval);
  }, [expiresAt, onExpire]);

  return (
    <div
      className={`text-center p-4 rounded-lg ${
        isExpired
          ? "bg-red-50 border border-red-200"
          : "bg-blue-50 border border-blue-200"
      }`}
    >
      <div className="text-sm text-gray-600 mb-1">
        {isExpired ? "Payment Expired" : "Time Remaining"}
      </div>
      <div
        className={`text-2xl font-bold ${
          isExpired ? "text-red-600" : "text-blue-600"
        }`}
      >
        {timeLeft}
      </div>
      {!isExpired && (
        <div className="text-xs text-gray-500 mt-1">
          Complete payment before expiry
        </div>
      )}
    </div>
  );
}
