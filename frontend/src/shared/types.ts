export interface User {
  id: number;
  username: string;
  email: string;
  firstName: string;
  lastName: string;
  roleCode: string;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  expiresAt: string;
  user: User;
}

export interface Team {
  teamId: string;
  name: string;
  avatar?: string;
  bindingGift: string;
  scoreMultipliers?: Record<string, number>;
  streamers: Streamer[];
}

export interface Streamer {
  streamerId: string;
  bigoId: string;
  bigoRoomId: string;
  name: string;
  avatar?: string;
  bindingGift: string;
}

export interface PKConfig {
  roomId: string;
  agencyId?: number;
  teams: Team[];
}

export interface SessionInfo {
  sessionId: string;
  status: string;
  startedAt: number;
  endsAt: number;
}
