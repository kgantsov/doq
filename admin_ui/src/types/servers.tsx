export type Suffrage = "Voter" | "Nonvoter" | "Staging";

export interface Server {
  id: string;
  addr: string;
  is_leader: boolean;
  suffrage: Suffrage;
}
