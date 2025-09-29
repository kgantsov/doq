export interface Queue {
  name: string;
  type: string;
  settings?: {
    max_unacked?: number;
    strategy?: string;
    ack_timeout?: string;
  };
  enqueue_rps: number;
  dequeue_rps: number;
  ack_rps: number;
  nack_rps: number;
  total: number;
  ready: number;
  unacked: number;
}
