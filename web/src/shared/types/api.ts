/**
 * TypeScript mirror of the Go backend DTOs (docs/openapi.yaml + internal/*).
 *
 * Privacy rule: fields like WorldBible, hidden_state, private_state and
 * internal_truth never appear here on purpose — the UI must never render them.
 */

export type MissionType =
  | "detective"
  | "wildlife_rescue"
  | "disaster_response"
  | "exploration"
  | "survival"
  | "diplomacy"
  | "medical_mystery";

export type Difficulty = "easy" | "medium" | "hard" | "expert";

export type MissionStatus =
  | "generating"
  | "ready"
  | "active"
  | "completed"
  | "failed"
  | "archived";

export type Language = "en" | "fa";

export interface APIErrorBody {
  code: string;
  message: string;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  access_expires_at: string;
  refresh_token: string;
  refresh_expires_at: string;
}

export interface AuthResponse {
  user: User;
  tokens: TokenPair;
}

export interface Profile {
  id: string;
  user_id: string;
  display_name: string;
  rank: string;
  level: number;
  xp: number;
  total_missions: number;
  completed_missions: number;
  failed_missions: number;
  success_rate: number;
  favorite_mission_type: string;
  total_clues_found: number;
  total_ai_interactions: number;
  total_locations_visited: number;
  badges: unknown;
  created_at: string;
  updated_at: string;
}

export interface ProfileStats {
  profile: Profile;
  total_coins_spent: number;
  total_coins_earned: number;
}

export interface HistoryEntry {
  mission_id: string;
  title: string;
  type: MissionType;
  difficulty: Difficulty;
  status: MissionStatus;
  result?: unknown;
  created_at: string;
  completed_at?: string | null;
}

export interface Wallet {
  id: string;
  user_id: string;
  balance: number;
  reserved_balance: number;
  created_at: string;
  updated_at: string;
}

export interface Transaction {
  id: string;
  wallet_id: string;
  user_id: string;
  mission_id?: string | null;
  type: string;
  amount: number;
  balance_after: number;
  metadata: unknown;
  created_at: string;
}

/** Server pricing rules: action name -> coin price. */
export type Pricing = Record<string, number>;

export interface Cost {
  coins_charged: number;
}

export interface Objective {
  id: string;
  title: string;
  description: string;
  status: "active" | "completed" | "failed";
  required_clues: number;
  optional: boolean;
}

export interface Mission {
  id: string;
  user_id: string;
  type: MissionType;
  title: string;
  status: MissionStatus;
  difficulty: Difficulty;
  region: string;
  summary: string;
  briefing: string;
  objectives: Objective[] | null;
  public_state: Record<string, unknown> | null;
  result?: unknown;
  center_lat: number;
  center_lng: number;
  map_zoom: number;
  current_time: string;
  created_at: string;
  updated_at: string;
  completed_at?: string | null;
}

export interface PublicCharacter {
  id: string;
  mission_id: string;
  name: string;
  role: string;
  category: string;
  age?: number;
  public_profile: string;
  personality: unknown;
  current_location_id?: string | null;
  trust_level: number;
  mood: string;
  avatar_prompt: string;
  thumbnail_prompt: string;
  visual_style_tags: unknown;
}

export interface InteractionMessage {
  id: string;
  interaction_id: string;
  sender: string;
  content: string;
  metadata: unknown;
  created_at: string;
}

export interface CharacterDetail {
  character: PublicCharacter;
  messages: InteractionMessage[];
}

export interface PublicClue {
  id: string;
  mission_id: string;
  location_id?: string | null;
  title: string;
  type: string;
  short_description: string;
  detailed_description: string;
  visual_description: string;
  avatar_or_thumbnail_prompt: string;
  discovered: boolean;
  reliability: number;
  importance: string;
  related_character_ids: unknown;
  public_data: Record<string, unknown> | null;
  created_at: string;
}

export interface Marker {
  id: string;
  name: string;
  type: string;
  lat: number;
  lng: number;
  status: string;
  risk_level: number;
  has_new_clue: boolean;
  has_character: boolean;
  is_locked: boolean;
  badge?: string;
}

export interface MapView {
  mission_id: string;
  center: { lat: number; lng: number };
  zoom: number;
  locations: Marker[];
}

export interface LocationEntity {
  id: string;
  mission_id: string;
  name: string;
  type: string;
  latitude: number;
  longitude: number;
  status: string;
  risk_level: number;
  description: string;
  visual_prompt: string;
  available_actions: string[] | null;
  created_at: string;
  updated_at: string;
}

export interface LocationDetail {
  location: LocationEntity;
  characters: PublicCharacter[];
  discovered_clues: PublicClue[];
}

export interface ActionResult {
  narrative: string;
  discovered_clues: PublicClue[];
  new_facts: string[];
  cost: Cost;
}

export interface ChatResult {
  message: string;
  emotion: string;
  mood: string;
  trust_level: number;
  trust_delta: number;
  stress_delta: number;
  unlocked_clues: PublicClue[];
  new_facts: string[];
  cost: Cost;
}

export interface ClueInspectResult {
  analysis: string;
  new_facts: string[];
  clue: PublicClue;
  cost: Cost;
}

export interface ClueExplainResult {
  explanation: string;
  next_steps: string[];
  compare_with: string[];
  cost: Cost;
}

export interface GuidanceItemRef {
  type: "location" | "clue" | "character" | "objective" | string;
  id: string;
  name: string;
}

export interface GuidanceResult {
  message: string;
  hint_level: string;
  referenced_items: GuidanceItemRef[];
  cost: Cost;
}

export interface GuidanceContext {
  screen: string;
  location_id?: string | null;
  selected_clue_id?: string | null;
}

export interface TimeInfo {
  current_time: string;
  public_state: Record<string, unknown> | null;
}

export interface TimeEvent {
  type: string;
  title: string;
}

export interface TimeAdvanceResult {
  new_time: string;
  summary: string;
  events: TimeEvent[];
  cost: Cost;
}

export type TimeUnit = "minutes" | "hours" | "days";

export interface JournalNote {
  id: string;
  mission_id: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface MissionEvent {
  id: string;
  mission_id: string;
  type: string;
  payload: Record<string, unknown>;
  created_at: string;
}

export interface MissionDashboard {
  mission: Mission;
  characters: PublicCharacter[];
  clues: PublicClue[];
  locations: Marker[];
}

export interface CreateMissionRequest {
  type: MissionType;
  difficulty: Difficulty;
  region?: string;
  language?: Language;
}

/** Base64 image attachment sent to vision-capable endpoints. */
export interface ImagePayload {
  data: string; // base64, no data: prefix
  mime: string; // image/png | image/jpeg | image/webp
}

export interface AvatarOptions {
  styles: string[];
  genders: string[];
  age_groups: string[];
}

export interface AvatarSpec {
  style?: string;
  gender?: string;
  age_group?: string;
  ethnicity?: string;
  seed?: string;
  size?: number;
}

export interface GeneratedAvatar {
  png_base64: string;
  mime: string;
  provider: string;
  style: string;
}
