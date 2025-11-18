import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Payment Gateway - Stablecoin Payments",
  description: "Secure crypto payment processing with USDT and USDC",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
      </body>
    </html>
  );
}
