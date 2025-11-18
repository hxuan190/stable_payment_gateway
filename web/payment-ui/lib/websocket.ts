import { PaymentEvent } from "@/types/payment";

const WS_BASE_URL =
  process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

type EventHandler = (event: PaymentEvent) => void;

export class PaymentWebSocket {
  private ws: WebSocket | null = null;
  private paymentId: string;
  private handlers: Map<string, EventHandler[]> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 2000;
  private heartbeatInterval: NodeJS.Timeout | null = null;

  constructor(paymentId: string) {
    this.paymentId = paymentId;
    this.connect();
  }

  private connect() {
    try {
      this.ws = new WebSocket(`${WS_BASE_URL}/ws/payments/${this.paymentId}`);

      this.ws.onopen = () => {
        console.log("WebSocket connected");
        this.reconnectAttempts = 0;
        this.startHeartbeat();
      };

      this.ws.onmessage = (event) => {
        try {
          const paymentEvent: PaymentEvent = JSON.parse(event.data);
          this.emit(paymentEvent.type, paymentEvent);
        } catch (error) {
          console.error("Failed to parse WebSocket message:", error);
        }
      };

      this.ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      this.ws.onclose = () => {
        console.log("WebSocket disconnected");
        this.stopHeartbeat();
        this.reconnect();
      };
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
      this.reconnect();
    }
  }

  private reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Max reconnect attempts reached");
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * this.reconnectAttempts;

    console.log(`Reconnecting in ${delay}ms... (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000); // 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  on(eventType: string, handler: EventHandler) {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, []);
    }
    this.handlers.get(eventType)!.push(handler);
  }

  off(eventType: string, handler: EventHandler) {
    const handlers = this.handlers.get(eventType);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  private emit(eventType: string, event: PaymentEvent) {
    const handlers = this.handlers.get(eventType);
    if (handlers) {
      handlers.forEach((handler) => handler(event));
    }
  }

  close() {
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.handlers.clear();
  }
}
