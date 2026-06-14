"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  Users, 
  Building2, 
  TrendingUp, 
  DollarSign, 
  Clock, 
  MessageCircle, 
  Star, 
  Shield,
  Zap,
  Brain,
  Target,
  ArrowRight,
  Check,
  Mail,
  Phone,
  Calendar,
  BarChart2,
  Trophy,
  Sparkles,
  RefreshCw,
  ChevronRight,
  ChevronLeft,
  Terminal,
  TrendingDown,
  AlertTriangle,
  Download,
  Phone as PhoneIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney } from "@/components/shared";

interface DesignPartner {
  id: string;
  name: string;
  company: string;
  role: string;
  logo: string;
  tier: "pro" | "enterprise";
  status: "active" | "onboarding" | "churned";
  joinedDate: string;
  monthlySpend: number;
  eventsPerMonth: number;
  acceptanceRate: number;
  npsScore: number;
  feedbackCount: number;
  lastActivity: string;
  featuresUsed: string[];
  healthScore: number;
}

interface FeedbackItem {
  id: string;
  partnerId: string;
  type: "feature-request" | "bug-report" | "improvement" | "praise";
  title: string;
  description: string;
  priority: "high" | "medium" | "low";
  status: "open" | "in-progress" | "done";
  createdAt: string;
  votes: number;
}

const MOCK_PARTNERS: DesignPartner[] = [
  {
    id: "partner_1",
    name: "Sarah Chen",
    company: "Vercel",
    role: "VP Engineering",
    logo: "VC",
    tier: "enterprise",
    status: "active",
    joinedDate: "2024-01-15",
    monthlySpend: 1247.50,
    eventsPerMonth: 89000,
    acceptanceRate: 0.94,
    npsScore: 9,
    feedbackCount: 12,
    lastActivity: "2 hours ago",
    featuresUsed: ["Cost Leaks", "Zombie Agents", "Prompt Graveyard", "Model Matrix", "Forecasting"],
    healthScore: 95,
  },
  {
    id: "partner_2",
    name: "Marcus Johnson",
    company: "Linear",
    role: "CTO",
    logo: "LN",
    tier: "pro",
    status: "active",
    joinedDate: "2024-02-20",
    monthlySpend: 423.20,
    eventsPerMonth: 34000,
    acceptanceRate: 0.87,
    npsScore: 8,
    feedbackCount: 8,
    lastActivity: "1 day ago",
    featuresUsed: ["Cost Leaks", "Model Matrix", "Forecasting"],
    healthScore: 88,
  },
  {
    id: "partner_3",
    name: "Priya Patel",
    company: "Notion",
    role: "Dir. AI Platform",
    logo: "NT",
    tier: "enterprise",
    status: "onboarding",
    joinedDate: "2024-05-10",
    monthlySpend: 2100.00,
    eventsPerMonth: 156000,
    acceptanceRate: 0.91,
    npsScore: 10,
    feedbackCount: 15,
    lastActivity: "30 min ago",
    featuresUsed: ["Cost Leaks", "Zombie Agents", "Prompt Graveyard", "Model Matrix", "Forecasting", "Custom Pricing"],
    healthScore: 92,
  },
  {
    id: "partner_4",
    name: "David Kim",
    company: "Retool",
    role: "Head of AI",
    logo: "RT",
    tier: "pro",
    status: "active",
    joinedDate: "2024-03-05",
    monthlySpend: 298.75,
    eventsPerMonth: 28000,
    acceptanceRate: 0.82,
    npsScore: 7,
    feedbackCount: 5,
    lastActivity: "4 hours ago",
    featuresUsed: ["Cost Leaks", "Model Matrix"],
    healthScore: 85,
  },
  {
    id: "partner_5",
    name: "Elena Rodriguez",
    company: "Stripe",
    role: "VP Product",
    logo: "ST",
    tier: "enterprise",
    status: "active",
    joinedDate: "2023-11-12",
    monthlySpend: 3450.00,
    eventsPerMonth: 280000,
    acceptanceRate: 0.96,
    npsScore: 10,
    feedbackCount: 22,
    lastActivity: "1 hour ago",
    featuresUsed: ["Cost Leaks", "Zombie Agents", "Prompt Graveyard", "Model Matrix", "Forecasting", "Custom Pricing", "RBAC", "Audit Trail"],
    healthScore: 98,
  },
  {
    id: "partner_6",
    name: "James Wilson",
    company: "Supabase",
    role: "Founder",
    logo: "SB",
    tier: "pro",
    status: "active",
    joinedDate: "2024-04-01",
    monthlySpend: 187.50,
    eventsPerMonth: 15000,
    acceptanceRate: 0.79,
    npsScore: 8,
    feedbackCount: 3,
    lastActivity: "6 hours ago",
    featuresUsed: ["Cost Leaks", "Model Matrix", "Forecasting"],
    healthScore: 82,
  },
];

const MOCK_FEEDBACK: FeedbackItem[] = [
  {
    id: "fb_1",
    partnerId: "partner_1",
    type: "feature-request",
    title: "Custom cost thresholds per model",
    description: "We need to set different cost alert thresholds for different models. GPT-4o should trigger at $50/day but Haiku at $5/day.",
    priority: "high",
    status: "in-progress",
    createdAt: "2024-05-20",
    votes: 8,
  },
  {
    id: "fb_2",
    partnerId: "partner_3",
    type: "feature-request",
    title: "Slack/Teams integration for alerts",
    description: "Real-time cost leak alerts should post to our #ai-ops channel with rich cards showing the exact pattern and estimated savings.",
    priority: "high",
    status: "open",
    createdAt: "2024-05-18",
    votes: 12,
  },
  {
    id: "fb_3",
    partnerId: "partner_5",
    type: "improvement",
    title: "RBAC for cost data visibility",
    description: "Finance team needs read-only access to cost dashboards without seeing prompt content or model configs.",
    priority: "medium",
    status: "done",
    createdAt: "2024-04-15",
    votes: 6,
  },
  {
    id: "fb_4",
    partnerId: "partner_2",
    type: "bug-report",
    title: "Zombie agent detection false positives on batch jobs",
    description: "Nightly batch processing agents flagged as zombies due to low acceptance rate, but they're intentionally processing large queues.",
    priority: "medium",
    status: "in-progress",
    createdAt: "2024-05-10",
    votes: 4,
  },
  {
    id: "fb_5",
    partnerId: "partner_4",
    type: "praise",
    title: "Prompt graveyard saved us $12k/month",
    description: "Identified 3 dead prompt templates that were costing us unnecessarily. ROI on TokenGoblin is insane.",
    priority: "low",
    status: "open",
    createdAt: "2024-05-05",
    votes: 15,
  },
  {
    id: "fb_6",
    partnerId: "partner_6",
    type: "feature-request",
    title: "Cost attribution by feature flag",
    description: "We run A/B tests with feature flags. Need to attribute costs to specific flag variants to measure experiment ROI.",
    priority: "medium",
    status: "open",
    createdAt: "2024-05-22",
    votes: 7,
  },
];

interface FeedbackItem {
  id: string;
  partnerId: string;
  type: "feature-request" | "bug-report" | "improvement" | "praise";
  title: string;
  description: string;
  priority: "high" | "medium" | "low";
  status: "open" | "in-progress" | "done";
  createdAt: string;
  votes: number;
}

const itemColors: Record<string, string> = {
  "VC": "#000000",
  "LN": "#56c4d6",
  "NT": "#000000",
  "RT": "#e06c6c",
  "ST": "#635bff",
  "SB": "#3ecf8e",
};

const typeIcons: Record<string, React.ComponentType<{ size?: number }>> = {
  "feature-request": Sparkles,
  "bug-report": Brain,
  "improvement": Target,
  "praise": Trophy,
};

const typeColors: Record<string, string> = {
  "feature-request": "text-blue-500",
  "bug-report": "text-red-500",
  "improvement": "text-orange-500",
  "praise": "text-green-500",
};

const priorityColors: Record<string, string> = {
  "high": "bg-red-900/30 text-red-400",
  "medium": "bg-yellow-900/30 text-yellow-400",
  "low": "bg-green-900/30 text-green-400",
};

export function DesignPartnerDashboard() {
  const [activeTab, setActiveTab] = useState<"overview" | "partners" | "feedback" | "health">("overview");
  const [selectedPartner, setSelectedPartner] = useState<DesignPartner | null>(null);

  const activePartners = MOCK_PARTNERS.filter(p => p.status === "active");
  const totalMRR = MOCK_PARTNERS.reduce((sum, p) => sum + p.monthlySpend, 0);
  const avgNPS = MOCK_PARTNERS.reduce((sum, p) => sum + p.npsScore, 0) / MOCK_PARTNERS.length;
  const avgHealth = MOCK_PARTNERS.reduce((sum, p) => sum + p.healthScore, 0) / MOCK_PARTNERS.length;
  const totalFeedback = MOCK_FEEDBACK.length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 p-4 bg-black border border-[#333] rounded-lg"
      >
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-[#ffb000] to-[#ff8c00] flex items-center justify-center">
            <Users size={24} className="text-black" />
          </div>
          <div>
            <h3 className="font-bold text-white uppercase tracking-widest text-lg">DESIGN PARTNER PROGRAM</h3>
            <p className="text-xs text-zinc-500 font-mono">{MOCK_PARTNERS.length} PARTNERS \u2022 ${Math.round(totalMRR).toLocaleString()} MRR</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setSelectedPartner(null)}>
            <Mail size={14} className="mr-1" />
            ANNOUNCEMENT
          </Button>
          <Button variant="outline" size="sm">
            <Calendar size={14} className="mr-1" />
            SCHEDULE CALL
          </Button>
          <Button size="sm" onClick={() => setActiveTab("partners")}>
            <Users size={14} className="mr-1" />
            ADD PARTNER
          </Button>
        </div>
      </motion.div>

      {/* Tabs */}
      <div className="flex border-b border-[#333]">
        {[
          { id: "overview", label: "OVERVIEW", icon: BarChart2 },
          { id: "partners", label: "PARTNERS", icon: Users },
          { id: "feedback", label: "FEEDBACK", icon: MessageCircle },
          { id: "health", label: "HEALTH", icon: Shield },
        ].map(tab => (
          <Button
            key={tab.id}
            variant={activeTab === tab.id ? "default" : "ghost"}
            size="sm"
            className="border-0 border-b-2 rounded-none hover:bg-transparent transition-colors flex items-center gap-2 uppercase tracking-widest text-xs"
            onClick={() => setActiveTab(tab.id as typeof activeTab)}
            style={{
              borderBottomColor: activeTab === tab.id ? "#ffb000" : "transparent",
              backgroundColor: activeTab === tab.id ? "#111" : "transparent",
            }}
          >
            <tab.icon size={14} />
            {tab.label}
          </Button>
        ))}
      </div>

      {/* Tab Content */}
      <AnimatePresence mode="wait">
        {activeTab === "overview" && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            key="overview"
          >
            <PartnerOverview />
          </motion.div>
        )}
        {activeTab === "partners" && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            key="partners"
          >
            <PartnersTable 
              partners={MOCK_PARTNERS} 
              onSelect={setSelectedPartner}
            />
          </motion.div>
        )}
        {activeTab === "feedback" && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            key="feedback"
          >
            <FeedbackBoard feedback={MOCK_FEEDBACK} />
          </motion.div>
        )}
        {activeTab === "health" && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            key="health"
          >
            <HealthDashboard partners={MOCK_PARTNERS} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Partner Detail Modal */}
      <AnimatePresence>
        {selectedPartner && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4"
            onClick={() => setSelectedPartner(null)}
          >
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: -20 }}
              className="w-full max-w-4xl max-h-[90vh] bg-black border border-[#333] rounded-xl overflow-hidden flex flex-col"
              onClick={e => e.stopPropagation()}
            >
              <PartnerDetailModal 
                partner={selectedPartner} 
                onClose={() => setSelectedPartner(null)}
              />
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function PartnerOverview() {
  return (
    <div className="space-y-6">
      {/* Key Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <OverviewMetric title="TOTAL PARTNERS" value="6" icon={Users} color="text-amber-500" trend="+2 this month" />
        <OverviewMetric title="MRR" value="$7,707" icon={DollarSign} color="text-green-500" trend="+$1,200" />
        <OverviewMetric title="AVG NPS" value="8.7" icon={Star} color="text-orange-500" trend="+0.3" />
        <OverviewMetric title="AVG HEALTH" value="90%" icon={Shield} color="text-cyan-500" trend="+5%" />
      </div>

      {/* Tier Distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="bg-black border-[#333]">
          <CardHeader>
            <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
              <Trophy size={16} className="text-amber-500" />
              ENTERPRISE TIER
            </CardTitle>
          </CardHeader>
          <CardContent>
            <TierStats partners={MOCK_PARTNERS.filter(p => p.tier === "enterprise")} color="text-orange-500" />
          </CardContent>
        </Card>

        <Card className="bg-black border-[#333]">
          <CardHeader>
            <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
              <Zap size={16} className="text-cyan-500" />
              PRO TIER
            </CardTitle>
          </CardHeader>
          <CardContent>
            <TierStats partners={MOCK_PARTNERS.filter(p => p.tier === "pro")} color="text-cyan-500" />
          </CardContent>
        </Card>

        <Card className="bg-black border-[#333]">
          <CardHeader>
            <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
              <Target size={16} className="text-green-500" />
              ONBOARDING
            </CardTitle>
          </CardHeader>
          <CardContent>
            <OnboardingPipeline />
          </CardContent>
        </Card>
      </div>

      {/* Timeline */}
      <Card className="bg-black border-[#333]">
        <CardHeader>
          <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
            <Clock size={16} className="text-cyan-500" />
            RECENT ACTIVITY
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ActivityTimeline />
        </CardContent>
      </Card>
    </div>
  );
}

function OverviewMetric({ title, value, icon: Icon, color, trend }: { 
  title: string; value: string; icon: React.ComponentType<{ size?: number; className?: string }>;
  color: string; trend: string;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="p-4 bg-black border border-[#333] rounded-lg"
    >
      <div className="flex items-center justify-between mb-2">
        <Icon size={18} className={color} />
        <span className="text-green-400 text-[10px] font-bold uppercase tracking-widest">{trend}</span>
      </div>
      <p className="text-zinc-500 text-[10px] uppercase tracking-widest mb-1">{title}</p>
      <p className="text-white font-bold text-2xl font-mono">{value}</p>
    </motion.div>
  );
}

function TierStats({ partners, color }: { partners: DesignPartner[]; color: string }) {
  const totalMRR = partners.reduce((sum, p) => sum + p.monthlySpend, 0);
  const totalEvents = partners.reduce((sum, p) => sum + p.eventsPerMonth, 0);
  const avgAcceptance = partners.reduce((sum, p) => sum + p.acceptanceRate, 0) / partners.length;
  
  return (
    <div className="space-y-4">
      <div className="flex items-baseline gap-2">
        <span className="text-3xl font-bold text-white font-mono">${totalMRR.toLocaleString()}</span>
        <span className="text-zinc-500 text-xs uppercase tracking-widest">MRR</span>
      </div>
      <div className="grid grid-cols-2 gap-4 text-sm">
        <MetricLine label="PARTNERS" value={String(partners.length)} color={color} />
        <MetricLine label="EVENTS/MO" value={totalEvents.toLocaleString()} color={color} />
        <MetricLine label="AVG ACCEPTANCE" value={(avgAcceptance * 100).toFixed(1) + "%"} color={color} />
        <MetricLine label="AVG NPS" value={String(partners.reduce((s, p) => s + p.npsScore, 0) / partners.length)} color={color} />
      </div>
      <div className="pt-4 border-t border-[#333]">
        <p className="text-zinc-500 text-xs uppercase tracking-widest mb-2">FEATURE ADOPTION</p>
        <FeatureAdoption partners={partners} />
      </div>
    </div>
  );
}

function MetricLine({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="p-3 bg-[#111] rounded border border-[#222]">
      <p className="text-zinc-500 text-[10px] uppercase tracking-widest mb-1">{label}</p>
      <p className="text-white font-bold" style={{ color }}>{value}</p>
    </div>
  );
}

function FeatureAdoption({ partners }: { partners: DesignPartner[] }) {
  const featureCounts: Record<string, number> = {};
  partners.forEach(p => {
    p.featuresUsed.forEach(f => {
      featureCounts[f] = (featureCounts[f] || 0) + 1;
    });
  });

  return (
    <div className="flex flex-wrap gap-2">
      {Object.entries(featureCounts)
        .sort(([,a], [,b]) => b - a)
        .slice(0, 8)
        .map(([feature, count]) => (
          <span key={feature} className="px-2 py-1 text-[10px] bg-[#111] border border-[#333] rounded text-zinc-400 font-mono uppercase tracking-widest">
            {feature} ({count})
          </span>
        ))}
    </div>
  );
}

function OnboardingPipeline() {
  const onboardingPartners = MOCK_PARTNERS.filter(p => p.status === "onboarding");
  
  return (
    <div className="space-y-4">
      {onboardingPartners.map(partner => (
        <motion.div
          key={partner.id}
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          className="p-4 bg-[#111] border border-[#333] rounded-lg"
        >
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-[#22c55e] to-[#16a34a] flex items-center justify-center text-white font-bold text-sm">
                {partner.logo}
              </div>
              <div>
                <p className="text-white font-bold text-sm">{partner.company}</p>
                <p className="text-zinc-500 text-xs">{partner.name} \u2022 {partner.role}</p>
              </div>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <span className="px-2 py-1 rounded-full bg-green-900/30 text-green-400 font-medium">ONBOARDING</span>
              <span className="text-zinc-500">Week {Math.ceil((Date.now() - new Date(partner.joinedDate).getTime()) / (7 * 86400000))}</span>
            </div>
          </div>
          <div className="flex items-center gap-2 mt-2">
            <div className="flex-1 h-2 bg-[#111] rounded-full overflow-hidden">
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${Math.min(100, (Date.now() - new Date(partner.joinedDate).getTime()) / (30 * 86400000) * 100)}%` }}
                className="h-full bg-green-500 rounded-full"
                transition={{ duration: 0.8 }}
              />
            </div>
            <span className="text-zinc-500 text-xs w-16 text-right">30 days</span>
          </div>
        </motion.div>
      ))}
      {onboardingPartners.length === 0 && (
        <p className="text-zinc-600 text-center py-8 text-xs uppercase">NO PARTNERS IN ONBOARDING</p>
      )}
    </div>
  );
}

function ActivityTimeline() {
  const activities = [
    { time: "2 hours ago", type: "feedback", text: "Sarah Chen (Vercel) requested custom cost thresholds", icon: MessageCircle, color: "#3b82f6" },
    { time: "4 hours ago", type: "signup", text: "Elena Rodriguez (Stripe) upgraded to Enterprise", icon: ArrowRight, color: "#22c55e" },
    { time: "1 day ago", type: "feedback", text: "Marcus Johnson (Linear) reported zombie agent false positive", icon: Brain, color: "#ef4444" },
    { time: "2 days ago", type: "milestone", text: "Priya Patel (Notion) completed onboarding", icon: Trophy, color: "#f97316" },
    { time: "3 days ago", type: "feedback", text: "David Kim (Retool) requested Slack integration", icon: Zap, color: "#f97316" },
    { time: "5 days ago", type: "signup", text: "James Wilson (Supabase) joined Pro tier", icon: Users, color: "#06b6d4" },
  ];

  return (
    <div className="space-y-4">
      {activities.map((activity, i) => (
        <motion.div
          key={i}
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: i * 0.1 }}
          className="flex items-start gap-3 p-3 bg-[#111] border border-[#222] rounded-lg hover:border-[#333] transition-colors"
        >
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: i * 0.1, type: "spring" }}
            className="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
            style={{ backgroundColor: activity.color + "20" }}
          >
            <activity.icon size={18} className={activity.color} />
          </motion.div>
          <div className="flex-1 min-w-0">
            <p className="text-white text-sm">{activity.text}</p>
            <p className="text-zinc-500 text-[10px] uppercase tracking-widest mt-1">{activity.time}</p>
          </div>
        </motion.div>
      ))}
    </div>
  );
}

function PartnersTable({ partners, onSelect }: { partners: DesignPartner[]; onSelect: (p: DesignPartner) => void }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-[#111] border-b border-[#333]">
          <tr>
            <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">PARTNER</th>
            <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">COMPANY</th>
            <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">TIER</th>
            <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">STATUS</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">MRR</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">EVENTS/MO</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">ACCEPTANCE</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">NPS</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">HEALTH</th>
            <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">LAST ACTIVE</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#222]">
          {partners.map((partner, i) => (
            <tr 
              key={partner.id} 
              className="hover:bg-[#0a0a0a] cursor-pointer transition-colors"
              onClick={() => onSelect(partner)}
            >
              <td className="px-4 py-3">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-[#ffb000] to-[#ff8c00] flex items-center justify-center text-black font-bold text-sm">
                    {partner.logo}
                  </div>
                  <div>
                    <p className="text-white font-bold text-sm">{partner.name}</p>
                    <p className="text-zinc-500 text-[10px] uppercase tracking-widest">{partner.role}</p>
                  </div>
                </div>
              </td>
              <td className="px-4 py-3">
                <p className="text-white font-mono text-sm">{partner.company}</p>
              </td>
              <td className="px-4 py-3">
                <span className={`px-2 py-1 text-[10px] rounded font-bold uppercase tracking-widest ${
                  partner.tier === "enterprise" 
                    ? "bg-amber-900/30 text-amber-400" 
                    : "bg-blue-900/30 text-blue-400"
                }`}>
                  {partner.tier.toUpperCase()}
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={`px-2 py-1 text-[10px] rounded font-bold uppercase tracking-widest ${
                  partner.status === "active" ? "bg-green-900/30 text-green-400" :
                  partner.status === "onboarding" ? "bg-green-900/30 text-green-400" :
                  "bg-red-900/30 text-red-400"
                }`}>
                  {partner.status.toUpperCase()}
                </span>
              </td>
              <td className="px-4 py-3 text-right font-bold text-white font-mono">${partner.monthlySpend.toLocaleString()}</td>
              <td className="px-4 py-3 text-right text-zinc-400 font-mono">{partner.eventsPerMonth.toLocaleString()}</td>
              <td className="px-4 py-3 text-right text-zinc-400 font-mono">{(partner.acceptanceRate * 100).toFixed(1)}%</td>
              <td className="px-4 py-3 text-right">
                <span className={`font-bold ${partner.npsScore >= 9 ? "text-green-400" : partner.npsScore >= 7 ? "text-yellow-400" : "text-red-400"}`}>
                  {partner.npsScore}/10
                </span>
              </td>
              <td className="px-4 py-3 text-right">
                <span className={`font-bold ${partner.healthScore >= 90 ? "text-green-400" : partner.healthScore >= 80 ? "text-yellow-400" : "text-red-400"}`}>
                  {partner.healthScore}%
                </span>
              </td>
              <td className="px-4 py-3 text-right text-zinc-500 text-xs">{partner.lastActivity}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function FeedbackBoard({ feedback }: { feedback: FeedbackItem[] }) {
  const columns = [
    { id: "open", title: "OPEN", items: feedback.filter(f => f.status === "open") },
    { id: "in-progress", title: "IN PROGRESS", items: feedback.filter(f => f.status === "in-progress") },
    { id: "done", title: "DONE", items: feedback.filter(f => f.status === "done") },
  ];

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {columns.map(column => (
        <Card key={column.id} className="bg-black border-[#333] h-full flex flex-col">
          <CardHeader className="border-b border-[#333] bg-[#0a0a0a">
            <div className="flex items-center justify-between">
              <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
                {column.title}
                <span className="px-2 py-0.5 text-[10px] bg-[#ffb000]20 text-[#ffb000] rounded font-bold">
                  {column.items.length}
                </span>
              </CardTitle>
            </div>
          </CardHeader>
          <CardContent className="flex-1 overflow-y-auto">
            <div className="space-y-3">
              {column.items.map((item, i) => {
                const Icon = typeIcons[item.type];
                return (
                  <motion.div
                    key={item.id}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.05 }}
                    className="p-4 bg-[#111] border border-[#222] rounded-lg hover:border-[#333] transition-colors"
                  >
                    <div className="flex items-start gap-3 mb-2">
                      <motion.div
                        initial={{ scale: 0 }}
                        animate={{ scale: 1 }}
                        className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
                        style={{ backgroundColor: typeColors[item.type] + "20" }}
                      >
                        <motion.span style={{ color: typeColors[item.type] }}>
                          <Icon size={18} />
                        </motion.span>
                      </motion.div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h5 className="text-white font-bold text-sm">{item.title}</h5>
                          <span className={`px-1.5 py-0.5 text-[10px] rounded font-bold uppercase tracking-widest ${typeColors[item.type].replace("text-", "bg-").replace("500", "900/30")}`}>
                            {item.type.toUpperCase().replace("-", " ")}
                          </span>
                          <span className={`px-1.5 py-0.5 text-[10px] rounded font-bold uppercase tracking-widest ${priorityColors[item.priority]}`}>
                            {item.priority.toUpperCase()}
                          </span>
                        </div>
                        <p className="text-zinc-400 text-xs mt-1 line-clamp-2">{item.description}</p>
                        <div className="flex items-center justify-between mt-2">
                          <span className="text-zinc-500 text-[10px] font-mono">{item.votes} VOTES</span>
                          <span className="text-zinc-500 text-[10px]">{item.createdAt}</span>
                        </div>
                      </div>
                    </div>
                  </motion.div>
                );
              })}
              {column.items.length === 0 && (
                <p className="text-zinc-600 text-center py-8 text-xs uppercase">NO FEEDBACK</p>
              )}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function HealthDashboard({ partners }: { partners: DesignPartner[] }) {
  return (
    <div className="space-y-6">
      {/* Health Distribution */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {[
          { label: "HEALTHY (\u226590%)", count: partners.filter(p => p.healthScore >= 90).length, color: "text-green-500" },
          { label: "AT RISK (80-89%)", count: partners.filter(p => p.healthScore >= 80 && p.healthScore < 90).length, color: "text-yellow-500" },
          { label: "CRITICAL (<80%)", count: partners.filter(p => p.healthScore < 80).length, color: "text-red-500" },
        ].map(item => (
          <Card key={item.label} className="bg-black border-[#333]">
            <CardHeader>
              <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center justify-between">
                {item.label}
                <span className="text-2xl font-bold" style={{ color: item.color }}>{item.count}</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="w-full h-2 bg-[#111] rounded-full overflow-hidden">
                <motion.div
                  initial={{ width: 0 }}
                  animate={{ width: `${(item.count / 6) * 100}%` }}
                  className="h-full rounded-full"
                  style={{ color: item.color }}
                  transition={{ duration: 0.8 }}
                />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Individual Health Scores */}
      <Card className="bg-black border-[#333]">
        <CardHeader>
          <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
            <Shield size={16} className="text-cyan-500" />
            INDIVIDUAL HEALTH SCORES
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-[#111] border-b border-[#333]">
                <tr>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">PARTNER</th>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">COMPANY</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">HEALTH</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">NPS</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">ACCEPTANCE</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">FEEDBACK</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">LAST ACTIVE</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#222]">
                {partners
                  .sort((a, b) => a.healthScore - b.healthScore)
                  .map((partner, i) => (
                    <tr key={partner.id} className="hover:bg-[#0a0a0a] transition-colors">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <div className="w-6 h-6 rounded flex items-center justify-center text-white font-bold text-xs" style={{ background: `linear-gradient(135deg, ${itemColors[partner.logo] || "#888"} 0%, ${itemColors[partner.logo] || "#888"}aa 100%)` }}>
                            {partner.logo}
                          </div>
                          <span className="text-white font-bold text-sm">{partner.name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-zinc-400 text-sm">{partner.company}</td>
                      <td className="px-4 py-3 text-center">
                        <span className={`px-3 py-1 rounded-full font-bold text-sm ${partner.healthScore >= 90 ? "bg-green-900/30 text-green-400" : partner.healthScore >= 80 ? "bg-yellow-900/30 text-yellow-400" : "bg-red-900/30 text-red-400"}`}>
                          {partner.healthScore}%
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span className={`font-bold ${partner.npsScore >= 9 ? "text-green-400" : partner.npsScore >= 7 ? "text-yellow-400" : "text-red-400"}`}>
                          {partner.npsScore}/10
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-zinc-400 font-mono">{(partner.acceptanceRate * 100).toFixed(1)}%</td>
                      <td className="px-4 py-3 text-center text-zinc-400 font-mono">{partner.feedbackCount}</td>
                      <td className="px-4 py-3 text-center text-zinc-500 text-xs">{partner.lastActivity}</td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function PartnerDetailModal({ partner, onClose }: { partner: DesignPartner; onClose: () => void }) {
  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-6 border-b border-[#333] bg-[#0a0a0a] flex items-center justify-between">
        <div className="flex items-center gap-4">
          <motion.button
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-[#111] transition-colors text-zinc-500 hover:text-white"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </motion.button>
          <div className="flex items-center gap-4">
            <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-[#ffb000] to-[#ff8c00] flex items-center justify-center text-black font-bold text-xl">
              {partner.logo}
            </div>
            <div>
              <h3 className="text-white font-bold text-xl uppercase tracking-widest">{partner.company}</h3>
              <p className="text-zinc-400 text-sm">{partner.name} \u2022 {partner.role}</p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <span className={`px-3 py-1 rounded-full font-bold text-sm ${partner.tier === "enterprise" ? "bg-amber-900/30 text-amber-400" : "bg-blue-900/30 text-blue-400"}`}>
            {partner.tier.toUpperCase()}
          </span>
          <span className={`px-3 py-1 rounded-full font-bold text-sm ${partner.status === "active" ? "bg-green-900/30 text-green-400" : partner.status === "onboarding" ? "bg-green-900/30 text-green-400" : "bg-red-900/30 text-red-400"}`}>
            {partner.status.toUpperCase()}
          </span>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Metrics Row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <DetailMetric title="MRR" value={`$${partner.monthlySpend.toLocaleString()}`} icon={DollarSign} color="text-green-500" />
          <DetailMetric title="EVENTS/MO" value={partner.eventsPerMonth.toLocaleString()} icon={Zap} color="text-amber-500" />
          <DetailMetric title="ACCEPTANCE" value={(partner.acceptanceRate * 100).toFixed(1) + "%"} icon={Check} color="text-cyan-500" />
          <DetailMetric title="NPS" value={partner.npsScore + "/10"} icon={Star} color="text-orange-500" />
          <DetailMetric title="HEALTH" value={partner.healthScore + "%"} icon={Shield} color={partner.healthScore >= 90 ? "text-green-500" : partner.healthScore >= 80 ? "text-yellow-500" : "text-red-500"} />
          <DetailMetric title="FEEDBACK" value={String(partner.feedbackCount)} icon={MessageCircle} color="text-purple-500" />
          <DetailMetric title="JOINED" value={new Date(partner.joinedDate).toLocaleDateString()} icon={Calendar} color="text-zinc-500" />
          <DetailMetric title="LAST ACTIVE" value={partner.lastActivity} icon={Clock} color="text-zinc-500" />
        </div>

        {/* Features Used */}
        <Card className="bg-black border-[#333]">
          <CardHeader>
            <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
              <Zap size={16} className="text-amber-500" />
              FEATURES USED ({partner.featuresUsed.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {partner.featuresUsed.map(feature => (
                <span key={feature} className="px-3 py-1.5 bg-[#111] border border-[#333] rounded-lg text-zinc-300 text-sm font-mono uppercase tracking-widest hover:border-[#ffb000] hover:text-[#ffb000] transition-colors">
                  {feature}
                </span>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Action Buttons */}
        <div className="flex gap-3 pt-4 border-t border-[#333]">
          <Button size="sm" onClick={onClose}>
            <Mail size={14} className="mr-1" />
            EMAIL PARTNER
          </Button>
          <Button variant="outline" size="sm" onClick={onClose}>
            <PhoneIcon size={14} className="mr-1" />
            SCHEDULE CALL
          </Button>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <MessageCircle size={14} className="mr-1" />
            VIEW FEEDBACK
          </Button>
        </div>
      </div>
    </div>
  );
}

function DetailMetric({ title, value, icon: Icon, color }: { title: string; value: string; icon: React.ComponentType<{ size?: number; className?: string }>; color: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="p-4 bg-[#111] border border-[#222] rounded-lg"
    >
      <div className="flex items-center justify-between mb-2">
        <Icon size={16} className={color} />
      </div>
      <p className="text-zinc-500 text-[10px] uppercase tracking-widest mb-1">{title}</p>
      <p className="text-white font-bold text-lg font-mono" style={{ color }}>{value}</p>
    </motion.div>
  );
}