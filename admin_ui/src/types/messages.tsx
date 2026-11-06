export interface Message {
  id: string;
  group: string;
  content: string;
  priority: number;
  delivery_count: number;
  status: number;
}
