"use client";

import { QRCodeCanvas } from "qrcode.react";
import { useState } from "react";

interface QRCodeProps {
  data: string;
  walletAddress: string;
  amount: string;
  memo: string;
  currency: string;
  chain: string;
}

export function QRCode({
  data,
  walletAddress,
  amount,
  memo,
  currency,
  chain,
}: QRCodeProps) {
  const [copied, setCopied] = useState<string | null>(null);

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="bg-white rounded-lg shadow-lg p-6">
      <div className="text-center mb-4">
        <h3 className="text-lg font-semibold mb-2">Scan to Pay</h3>
        <p className="text-sm text-gray-600">
          Scan this QR code with your {chain === "solana" ? "Solana" : "BSC"}{" "}
          wallet
        </p>
      </div>

      <div className="flex justify-center mb-6">
        <div className="bg-white p-4 rounded-lg border-2 border-gray-200">
          <QRCodeCanvas value={data} size={240} level="M" />
        </div>
      </div>

      <div className="space-y-3">
        <div className="bg-gray-50 p-3 rounded">
          <div className="flex justify-between items-center">
            <div className="flex-1 mr-2">
              <div className="text-xs text-gray-600 mb-1">Wallet Address</div>
              <div className="text-sm font-mono break-all">
                {walletAddress}
              </div>
            </div>
            <button
              onClick={() => copyToClipboard(walletAddress, "address")}
              className="px-3 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600 whitespace-nowrap"
            >
              {copied === "address" ? "Copied!" : "Copy"}
            </button>
          </div>
        </div>

        <div className="bg-gray-50 p-3 rounded">
          <div className="flex justify-between items-center">
            <div className="flex-1 mr-2">
              <div className="text-xs text-gray-600 mb-1">Amount</div>
              <div className="text-sm font-mono">
                {amount} {currency}
              </div>
            </div>
            <button
              onClick={() => copyToClipboard(amount, "amount")}
              className="px-3 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600 whitespace-nowrap"
            >
              {copied === "amount" ? "Copied!" : "Copy"}
            </button>
          </div>
        </div>

        <div className="bg-gray-50 p-3 rounded">
          <div className="flex justify-between items-center">
            <div className="flex-1 mr-2">
              <div className="text-xs text-gray-600 mb-1">Payment Reference</div>
              <div className="text-sm font-mono break-all">{memo}</div>
            </div>
            <button
              onClick={() => copyToClipboard(memo, "memo")}
              className="px-3 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600 whitespace-nowrap"
            >
              {copied === "memo" ? "Copied!" : "Copy"}
            </button>
          </div>
        </div>
      </div>

      <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded">
        <p className="text-xs text-yellow-800">
          <strong>Important:</strong> Make sure to include the payment reference
          in the transaction memo field.
        </p>
      </div>
    </div>
  );
}
