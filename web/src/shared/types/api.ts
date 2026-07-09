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

export interface CoinPack {
  product_id: string;
  coins: number;
}

export interface WalletConfig {
  /** Whether mock purchase/ad crediting is available in this environment. */
  demo_purchases: boolean;
  coin_packs: CoinPack[];
  rewarded_ad_coins: number;
}

export interface Cost {
  coins_charged: number;
}

export type ObjectiveType =
  | "primary"
  | "required"
  | "optional"
  | "hidden"
  | "dynamic"
  | "final";

export interface Objective {
  id: string;
  /** Added by the mission-guidance backend upgrade; older data omits it. */
  type?: ObjectiveType;
  title: string;
  description: string;
  status: "locked" | "active" | "completed" | "failed" | "skipped";
  /** 0-100; older data omits it. */
  progress?: number;
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
  /** Visual asset contract — generation prompts never cross this boundary. */
  avatar_url: string;
  avatar_status: "none" | "pending" | "ready" | "unavailable";
  avatar_version?: number;
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
  /** Visual asset contract — generation prompts never cross this boundary. */
  image_url: string;
  image_status: "none" | "pending" | "ready" | "unavailable";
  image_version?: number;
  discovered: boolean;
  /** Evidence lifecycle stage (added by the gameplay-state fix). */
  status?: "discovered" | "inspected" | "confirmed";
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
  /** Recommendation fields added by the mission-guidance backend upgrade. */
  objective_status?: string;
  has_required_action?: boolean;
  recommended?: boolean;
  priority?: string;
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
  available_actions: string[] | null;
  created_at: string;
  updated_at: string;
}

export interface LocationDetail {
  location: LocationEntity;
  characters: PublicCharacter[];
  discovered_clues: PublicClue[];
  /** Set when this call was a first visit (travel costs mission time). */
  time_update?: TimeUpdate | null;
}

export interface ActionResult {
  narrative: string;
  discovered_clues: PublicClue[];
  new_facts: string[];
  cost: Cost;
  time_update?: TimeUpdate | null;
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
  time_update?: TimeUpdate | null;
}

export interface ClueInspectResult {
  analysis: string;
  new_facts: string[];
  clue: PublicClue;
  cost: Cost;
  time_update?: TimeUpdate | null;
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

export interface CompletionCheck {
  can_complete: boolean;
  reason: string;
  missing_requirements: string[];
}

export interface MissionResult {
  can_complete: boolean;
  success: boolean;
  score: number;
  stars: number;
  result_title: string;
  result_summary: string;
  completed_objectives: string[];
  failed_objectives: string[];
  missed_optional_objectives: string[];
  critical_clues_found: string[];
  critical_clues_missed: string[];
  good_decisions: string[];
  bad_decisions: string[];
  xp_reward: number;
  coin_reward: number;
  cost: Cost;
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

/** Curated player-facing timeline item types (GET /missions/{id}/timeline). */
export type TimelineItemType =
  | "mission_started"
  | "location_visited"
  | "clue_discovered"
  | "clue_inspected"
  | "evidence_confirmed"
  | "hypothesis_submitted"
  | "character_talked"
  | "ai_guidance_received"
  | "time_advanced"
  | "risk_changed"
  | "objective_completed"
  | "objective_failed"
  | "new_location_unlocked"
  | "mission_ready_to_complete"
  | "mission_completed"
  | "mission_failed"
  | "world_event";

export interface TimelineItem {
  id: string;
  type: TimelineItemType;
  title: string;
  description?: string;
  mission_time?: string;
  occurred_at: string;
  location_id?: string | null;
  related_clue_id?: string | null;
  related_character_id?: string | null;
  importance: "high" | "medium" | "low";
}

export interface TimelineView {
  mission_id: string;
  items: TimelineItem[];
}

/** Uniform envelope returned by state-changing progression actions. */
export interface ProgressionEnvelope {
  message: string;
  state_changes: Array<{ entity: string; id: string; from: string; to: string }>;
  timeline_events: string[];
  unlocked_locations: Array<{ id: string; name: string; reason: string }>;
  next_recommended_actions: Array<{ type: string; title: string }>;
  stage_update?: StageUpdate | null;
}

export type HypothesisVerdict =
  | "too_early"
  | "unsupported"
  | "partially_correct";

export interface HypothesisResult {
  verdict: HypothesisVerdict;
  feedback: string;
  needs_more_evidence: boolean;
  confirmed_count: number;
  progression: ProgressionEnvelope;
}

export interface RecommendedAction {
  type: string;
  title: string;
  description: string;
  target_type?: string;
  target_id?: string;
  priority?: string;
  cost_hint?: string;
}

export interface MissionGuidanceSummary {
  summary?: string;
  warning?: string;
  recommended_actions?: RecommendedAction[];
}

/** Living-world snapshot that drives dynamic UI themes (weather/time/danger). */
export interface WorldState {
  mission_time: string;
  weather: "clear" | "rain" | "snow" | "storm" | "fog";
  time_of_day: "day" | "dusk" | "night";
  visibility: "high" | "medium" | "low";
  risk_score: number;
  urgency: "calm" | "rising" | "critical";
  world_phase: "opening" | "investigation" | "closing";
  danger: boolean;
  theme_id: string;
  active_events: string[];
}

export interface MissionDashboard {
  mission: Mission;
  world_state?: WorldState;
  mission_id?: string;
  title?: string;
  mission_status?: MissionStatus;
  primary_objective?: Objective | null;
  objectives?: Objective[];
  completed_objectives?: Objective[];
  mission_progress?: number;
  risk_score?: number;
  current_time?: string;
  time_remaining?: string;
  has_deadline?: boolean;
  next_recommended_actions?: RecommendedAction[];
  guidance?: MissionGuidanceSummary;
  win_conditions?: string[];
  failure_conditions?: string[];
  can_complete?: boolean;
  missing_requirements?: string[];
  characters: PublicCharacter[];
  clues: PublicClue[];
  locations: Marker[];
  timeline_preview?: MissionEvent[];
  wallet_balance?: number;
  result?: unknown;
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

/* --- Playable mission: stages, reports, time costs, board art --- */

export type StageStatus = "locked" | "active" | "completed";

export type StageActionType =
  | "visit_locations"
  | "find_clues"
  | "confirm_evidence"
  | "interview_characters"
  | "submit_report"
  | "final_decision";

export interface StageAction {
  type: StageActionType;
  report?: string;
  count: number;
  done: number;
}

export interface StageReward {
  xp: number;
  coins: number;
  badge?: string;
}

export interface StageUnlock {
  next_stage_id?: string;
  unlock_location?: boolean;
  reveal_suspect?: boolean;
}

export interface Stage {
  id: string;
  title: string;
  description: string;
  status: StageStatus;
  progress: number;
  required_clue_count: number;
  found_clue_count: number;
  required_actions: StageAction[];
  reward: StageReward;
  unlock_on_complete: StageUnlock;
}

/** Everything a stage-engine pass changed (drives popups/reveals). */
export interface StageUpdate {
  stages: Stage[];
  current_stage?: Stage | null;
  stage_index: number;
  stage_count: number;
  completed_stages: Stage[];
  activated_stage?: Stage | null;
  rewards: StageReward[];
  unlocked_locations: Array<{ id: string; name: string; reason: string }>;
  suspect_revealed: boolean;
  suspect?: PublicCharacter | null;
  timeline_events: string[];
}

/** Time passed + any world events the passage of time triggered. */
export interface TimeUpdate {
  new_time: string;
  minutes_advanced: number;
  triggered_events: Array<{
    type: string;
    title: string;
    description?: string;
  }>;
}

export interface BoardImage {
  url: string;
  status: string;
  version: number;
}

export type BoardType =
  | "mission_board_background"
  | "map_board_background"
  | "report_center_background"
  | "debrief_background"
  | "loading_screen";

export type SuspectStatus = "hidden" | "identified";

/** GET /missions/{id}/gameplay-status — the single payload the HUD lives on. */
export interface GameplayStatus extends MissionDashboard {
  stages: Stage[];
  current_stage?: Stage | null;
  stage_index: number;
  stage_count: number;
  clue_goal: number;
  clues_found: number;
  suspect_status: SuspectStatus;
  suspect?: PublicCharacter | null;
  next_reward?: StageReward | null;
  report_pending: boolean;
  pending_report_type?: string;
  board_art: Record<string, BoardImage>;
  stage_update?: StageUpdate | null;
}

export interface ActionPreview {
  action: string;
  time_cost_minutes: number;
  coin_cost: number;
  risk_note: string;
  new_time_if_done: string;
}

export type ReportType =
  | "clue_report"
  | "suspect_report"
  | "progress_report"
  | "incident_report"
  | "final_report";

export type ReportVerdict = "accepted" | "rejected";

export interface MissionReport {
  id: string;
  mission_id: string;
  type: ReportType;
  title: string;
  summary: string;
  linked_clue_ids: string[] | null;
  suspect_character_id?: string | null;
  verdict: ReportVerdict;
  feedback: string;
  created_at: string;
}

export interface ReportResult {
  verdict: ReportVerdict;
  feedback: string;
  missing_requirements: string[];
  report: MissionReport;
  stage_update?: StageUpdate | null;
  time_update?: TimeUpdate | null;
  timeline_events: string[];
  next_recommended_actions: Array<{ type: string; title: string }>;
}
