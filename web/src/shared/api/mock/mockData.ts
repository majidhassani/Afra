import type {
  JournalNote,
  Marker,
  Mission,
  MissionEvent,
  PublicCharacter,
  PublicClue,
} from "@/shared/types/api";

/** Deterministic-ish id generator for mock entities. */
export function mockId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `mock-${Math.random().toString(36).slice(2)}`;
}

/** Mirrors the real backend pricing keys (GET /api/v1/wallet/pricing). */
export const MOCK_PRICING: Record<string, number> = {
  mission_start: 150,
  character_chat: 3,
  clue_inspect: 3,
  clue_explain: 2,
  ai_guidance: 2,
  location_search: 5,
  location_ask: 2,
  advance_time: 4,
  final_judgment: 20,
};

const now = () => new Date().toISOString();

export interface MockMissionBundle {
  mission: Mission;
  markers: Marker[];
  locationDescriptions: Record<
    string,
    { description: string; actions: string[] }
  >;
  characters: PublicCharacter[];
  clues: PublicClue[];
  events: MissionEvent[];
  notes: JournalNote[];
}

/** Build a fully-populated sample mission in the requested language. */
export function buildMissionBundle(
  missionId: string,
  userId: string,
  type: Mission["type"],
  difficulty: Mission["difficulty"],
  region: string,
  language: "en" | "fa",
): MockMissionBundle {
  const fa = language === "fa";
  const centerLat = 35.7219;
  const centerLng = 51.3347;

  const mission: Mission = {
    id: missionId,
    user_id: userId,
    type,
    title: fa ? "پرونده کلاغ خاموش" : "The Case of the Silent Crow",
    status: "active",
    difficulty,
    region: region || (fa ? "تهران" : "Tehran"),
    summary: fa
      ? "یک پژوهشگر آرشیو ناپدید شده و تنها ردی که مانده، یک دفترچه نیمه‌سوخته است."
      : "An archive researcher has vanished; the only trace left is a half-burned notebook.",
    briefing: fa
      ? "دکتر آرش کاویانی، پژوهشگر آرشیو ملی، سه روز پیش ناپدید شد. آخرین بار در کتابخانه مرکزی دیده شده است. دفترچه نیمه‌سوخته‌ای در دفترش پیدا شده که به یک مجموعه خصوصی اشاره می‌کند. ماموریت شما: مسیر او را بازسازی کنید، با شاهدان صحبت کنید و بفهمید چه اتفاقی افتاده است."
      : "Dr. Arash Kaviani, a national archive researcher, disappeared three days ago. He was last seen at the Central Library. A half-burned notebook found in his office points to a private collection. Your mission: reconstruct his path, interview witnesses, and find out what happened.",
    objectives: [
      {
        id: "obj-1",
        title: fa ? "بازسازی ۲۴ ساعت آخر" : "Reconstruct the last 24 hours",
        description: fa
          ? "مسیر حرکت کاویانی را پیش از ناپدید شدن مشخص کنید."
          : "Establish Kaviani's movements before he vanished.",
        status: "active",
        required_clues: 3,
        optional: false,
      },
      {
        id: "obj-2",
        title: fa ? "شناسایی تماس ناشناس" : "Identify the anonymous caller",
        description: fa
          ? "شماره‌ای که سه بار با او تماس گرفته را ردیابی کنید."
          : "Trace the number that called him three times.",
        status: "active",
        required_clues: 2,
        optional: false,
      },
      {
        id: "obj-3",
        title: fa ? "یافتن مجموعه خصوصی" : "Locate the private collection",
        description: fa
          ? "مجموعه‌ای که در دفترچه به آن اشاره شده را پیدا کنید."
          : "Find the collection referenced in the notebook.",
        status: "active",
        required_clues: 2,
        optional: true,
      },
    ],
    public_state: fa
      ? { وضعیت: "پلیس پرونده را باز نگه داشته است", جو: "بارانی و سرد" }
      : { police: "case remains open", weather: "cold rain over the city" },
    center_lat: centerLat,
    center_lng: centerLng,
    map_zoom: 13,
    current_time: "Day 1 — 09:00",
    created_at: now(),
    updated_at: now(),
  };

  const locIds = [mockId(), mockId(), mockId(), mockId(), mockId()];

  const markers: Marker[] = [
    {
      id: locIds[0],
      name: fa ? "کتابخانه مرکزی" : "Central Library",
      type: "library",
      lat: centerLat + 0.008,
      lng: centerLng - 0.01,
      status: "discovered",
      risk_level: 1,
      has_new_clue: true,
      has_character: true,
      is_locked: false,
      badge: "start",
    },
    {
      id: locIds[1],
      name: fa ? "دفتر آرشیو" : "Archive Office",
      type: "office",
      lat: centerLat - 0.006,
      lng: centerLng + 0.012,
      status: "discovered",
      risk_level: 2,
      has_new_clue: true,
      has_character: false,
      is_locked: false,
    },
    {
      id: locIds[2],
      name: fa ? "کافه شمعدونی" : "Café Shamdooni",
      type: "cafe",
      lat: centerLat + 0.002,
      lng: centerLng + 0.02,
      status: "visited",
      risk_level: 1,
      has_new_clue: false,
      has_character: true,
      is_locked: false,
    },
    {
      id: locIds[3],
      name: fa ? "انبار متروک" : "Abandoned Warehouse",
      type: "warehouse",
      lat: centerLat - 0.012,
      lng: centerLng - 0.016,
      status: "discovered",
      risk_level: 4,
      has_new_clue: false,
      has_character: false,
      is_locked: true,
    },
    {
      id: locIds[4],
      name: fa ? "خانه کاویانی" : "Kaviani Residence",
      type: "residence",
      lat: centerLat + 0.014,
      lng: centerLng + 0.004,
      status: "discovered",
      risk_level: 2,
      has_new_clue: false,
      has_character: false,
      is_locked: false,
    },
  ];

  const locationDescriptions: MockMissionBundle["locationDescriptions"] = {
    [locIds[0]]: {
      description: fa
        ? "سالن مطالعه با سقف بلند و بوی کاغذ قدیمی. میز شماره ۱۲ همان‌جایی است که کاویانی آخرین بار دیده شد."
        : "A high-ceilinged reading hall smelling of old paper. Desk 12 is where Kaviani was last seen.",
      actions: ["inspect_area", "review_documents", "observe"],
    },
    [locIds[1]]: {
      description: fa
        ? "دفتری کوچک و شلوغ. کشوی میز قفل شده و اثر سوختگی روی سطل زباله دیده می‌شود."
        : "A small cluttered office. The desk drawer is locked; scorch marks stain the wastebasket.",
      actions: ["inspect_area", "scan_environment", "review_documents"],
    },
    [locIds[2]]: {
      description: fa
        ? "کافه‌ای دنج که پاتوق پژوهشگران است. باریستا همه را می‌شناسد."
        : "A cozy café favored by researchers. The barista knows everyone.",
      actions: ["observe", "inspect_area"],
    },
    [locIds[3]]: {
      description: fa
        ? "انباری در حاشیه شهر. درِ آن با زنجیر تازه بسته شده است."
        : "A warehouse on the city's edge. Its door is chained with a brand-new lock.",
      actions: ["scan_environment", "observe"],
    },
    [locIds[4]]: {
      description: fa
        ? "آپارتمانی مرتب اما عجیب: چمدانی نیمه‌بسته روی تخت مانده است."
        : "A tidy apartment with one oddity: a half-packed suitcase left on the bed.",
      actions: ["inspect_area", "review_documents", "search"],
    },
  };

  const charIds = [mockId(), mockId(), mockId()];

  const characters: PublicCharacter[] = [
    {
      id: charIds[0],
      mission_id: missionId,
      name: fa ? "لیلا فرهمند" : "Leila Farahmand",
      role: fa ? "کتابدار ارشد" : "Head Librarian",
      category: "witness",
      age: 48,
      public_profile: fa
        ? "بیست سال است در کتابخانه کار می‌کند. دقیق و کم‌حرف، اما هیچ چیز از چشمش پنهان نمی‌ماند."
        : "Twenty years at the library. Precise and reserved, but nothing escapes her.",
      personality: { traits: ["observant", "guarded"] },
      current_location_id: locIds[0],
      trust_level: 40,
      mood: "wary",
      avatar_url: "",
      avatar_status: "none",
      visual_style_tags: ["cinematic", "muted"],
    },
    {
      id: charIds[1],
      mission_id: missionId,
      name: fa ? "بهرام توسلی" : "Bahram Tavassoli",
      role: fa ? "همکار پژوهشگر" : "Fellow Researcher",
      category: "person_of_interest",
      age: 39,
      public_profile: fa
        ? "آخرین نفری که با کاویانی حرف زده. عصبی به نظر می‌رسد و مدام ساعتش را چک می‌کند."
        : "The last person who spoke with Kaviani. He seems nervous and keeps checking his watch.",
      personality: { traits: ["anxious", "evasive"] },
      current_location_id: locIds[2],
      trust_level: 25,
      mood: "nervous",
      avatar_url: "",
      avatar_status: "none",
      visual_style_tags: ["cinematic"],
    },
    {
      id: charIds[2],
      mission_id: missionId,
      name: fa ? "سارا نادری" : "Sara Naderi",
      role: fa ? "باریستا" : "Barista",
      category: "witness",
      age: 27,
      public_profile: fa
        ? "حافظه‌ای عالی برای چهره‌ها دارد. کاویانی مشتری ثابت صبح‌ها بود."
        : "Has a perfect memory for faces. Kaviani was a regular every morning.",
      personality: { traits: ["friendly", "talkative"] },
      current_location_id: locIds[2],
      trust_level: 60,
      mood: "helpful",
      avatar_url: "",
      avatar_status: "none",
      visual_style_tags: ["warm"],
    },
  ];

  const clueIds = [mockId(), mockId(), mockId()];

  const clues: PublicClue[] = [
    {
      id: clueIds[0],
      mission_id: missionId,
      location_id: locIds[1],
      title: fa ? "دفترچه نیمه‌سوخته" : "Half-burned Notebook",
      type: "document",
      short_description: fa
        ? "دفترچه‌ای که کسی سعی کرده نابودش کند."
        : "A notebook someone tried to destroy.",
      detailed_description: fa
        ? "صفحات باقی‌مانده به «مجموعه م.» و یک قرار در روز پنجشنبه اشاره می‌کنند. خط آخر ناتمام است."
        : 'The surviving pages mention "the M. collection" and a Thursday meeting. The last line is unfinished.',
      visual_description: fa
        ? "دفترچه چرمی با لبه‌های سوخته و جوهر آبی"
        : "Leather notebook with charred edges and blue ink",
      image_url: "",
      image_status: "none",
      discovered: true,
      reliability: 80,
      importance: "critical",
      related_character_ids: [charIds[1]],
      public_data: { pages_left: 14 },
      created_at: now(),
    },
    {
      id: clueIds[1],
      mission_id: missionId,
      location_id: locIds[0],
      title: fa ? "برگه امانت کتاب" : "Library Loan Slip",
      type: "document",
      short_description: fa
        ? "آخرین کتابی که کاویانی امانت گرفت."
        : "The last book Kaviani checked out.",
      detailed_description: fa
        ? "فهرست اموال یک خانواده قدیمی، چاپ ۱۳۲۹. برگه ساعت ۱۸:۴۰ پنجشنبه مهر خورده است."
        : "An estate inventory of an old family, printed 1950. Stamped Thursday 18:40.",
      visual_description: fa
        ? "برگه زردشده با مهر تاریخ"
        : "Yellowed slip with a date stamp",
      image_url: "",
      image_status: "none",
      discovered: true,
      reliability: 95,
      importance: "high",
      related_character_ids: [charIds[0]],
      public_data: { book: fa ? "فهرست اموال" : "Estate Inventory, 1950" },
      created_at: now(),
    },
    {
      id: clueIds[2],
      mission_id: missionId,
      location_id: locIds[2],
      title: fa ? "رسید کافه" : "Café Receipt",
      type: "physical",
      short_description: fa
        ? "دو فنجان، یک صندلی خالی."
        : "Two cups, one empty chair.",
      detailed_description: fa
        ? "رسید پنجشنبه ساعت ۱۹:۱۵ برای دو اسپرسو. کاویانی همیشه تنها می‌آمد."
        : "Thursday 19:15, two espressos. Kaviani always came alone.",
      visual_description: fa
        ? "رسید مچاله با ساعت چاپ‌شده"
        : "Crumpled receipt with printed time",
      image_url: "",
      image_status: "none",
      discovered: true,
      reliability: 70,
      importance: "medium",
      related_character_ids: [charIds[2]],
      public_data: { items: 2 },
      created_at: now(),
    },
  ];

  const events: MissionEvent[] = [
    {
      id: mockId(),
      mission_id: missionId,
      type: "mission_generated",
      payload: { title: mission.title },
      created_at: now(),
    },
    {
      id: mockId(),
      mission_id: missionId,
      type: "location_discovered",
      payload: { name: markers[0].name },
      created_at: now(),
    },
    {
      id: mockId(),
      mission_id: missionId,
      type: "clue_discovered",
      payload: { title: clues[0].title },
      created_at: now(),
    },
  ];

  const notes: JournalNote[] = [
    {
      id: mockId(),
      mission_id: missionId,
      title: fa ? "فرضیه اول" : "First hypothesis",
      content: fa
        ? "دو فنجان یعنی کاویانی با کسی قرار داشت. توسلی؟"
        : "Two cups means Kaviani met someone. Tavassoli?",
      created_at: now(),
      updated_at: now(),
    },
  ];

  return {
    mission,
    markers,
    locationDescriptions,
    characters,
    clues,
    events,
    notes,
  };
}
