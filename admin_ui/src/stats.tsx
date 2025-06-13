export interface QueueStats {
  name: string;
  type: string;
  settings?: {
    max_unacked?: number;
    strategy?: string;
  };
  enqueue_rps: number;
  dequeue_rps: number;
  ack_rps: number;
  nack_rps: number;
  ready: number;
  unacked: number;
  total: number;
  timestamp: number; // added to track when the point was added
}

export class QueueStatsBuffer {
  private buffer: QueueStats[] = [];
  private maxSize: number;

  constructor(maxSize: number = 300) {
    this.maxSize = maxSize;
  }

  append(stat: Omit<QueueStats, "timestamp">) {
    const point: QueueStats = {
      ...stat,
      timestamp: Date.now(),
    };

    if (this.buffer.length >= this.maxSize) {
      this.buffer.shift(); // Remove the oldest
    }
    this.buffer.push(point);
  }

  getPoints(): QueueStats[] {
    return this.buffer;
  }

  clear() {
    this.buffer = [];
  }
}
