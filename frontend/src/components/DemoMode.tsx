"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  Play, 
  Pause, 
  RefreshCw, 
  Zap, 
  TrendingUp, 
  TrendingDown, 
  AlertTriangle,
  DollarSign,
  Brain,
  Terminal,
  ArrowRight,
  Check,
  X,
  Sparkles,
  Download
} from "lucide-react";

import { formatMoney } from "@/components/shared";

interface DemoEvent {
  id: string;
  timestamp: string;
  worker: string;
  model: string;
  provider: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  cost: number;
  category: string;
  status: "accepted" | "rejected" | "flagged";
  pattern?: string;
}

interface DemoMetrics {
  totalEvents: number;
  totalCost: number;
  totalTokens: number;
  acceptanceRate: number;
  costLeaks: number;
  zombieAgents: number;
  topModel: string;
  wastePercent: number;
}

const DEMO_CATEGORIES = [
  "code_generation",
  "code_review", 
  "documentation",
  "testing",
  "debugging",
  "architecture",
  "refactoring",
  "security_audit",
];

const DEMO_MODELS = [
  { id: "gpt-4o", provider: "OpenAI", costPer1k: { input: 5.00, output: 15.00 } },
  { id: "gpt-4o-mini", provider: "OpenAI", costPer1k: { input: 0.15, output: 0.60 } },
  { id: "claude-3-5-sonnet", provider: "Anthropic", costPer1k: { input: 3.00, output: 15.00 } },
  { id: "claude-3-haiku", provider: "Anthropic", costPer1k: { input: 0.25, output: 1.25 } },
];

const DEMO_WORKERS = [
  "frontend-agent", "backend-agent", "review-agent", "test-agent", 
  "doc-agent", "security-agent", "refactor-agent", "arch-agent"
];

function generateDemoEvents(count: number): DemoEvent[] {
  const events: DemoEvent[] = [];
  const baseTime = Date.now() - 3600000; // 1 hour ago
  
  for (let i = 0; i < count; i++) {
    const model = DEMO_MODELS[Math.floor(Math.random() * DEMO_MODELS.length)];
    const worker = DEMO_WORKERS[Math.floor(Math.random() * DEMO_WORKERS.length)];
    const category = DEMO_CATEGORIES[Math.floor(Math.random() * DEMO_CATEGORIES.length)];
    const promptTokens = Math.floor(Math.random() * 8000) + 500;
    const completionTokens = Math.floor(Math.random() * 4000) + 200;
    const totalTokens = promptTokens + completionTokens;
    const cost = (promptTokens / 1000 * model.costPer1k.input) + (completionTokens / 1000 * model.costPer1k.output);
    const status = Math.random() > 0.85 ? "flagged" : (Math.random() > 0.15 ? "accepted" : "rejected");
    
    const patterns = [
      "over_tokening", "hallucination_loop", "redundant_call", 
      "zombie_agent", "token_bloat", "retry_storm"
    ];
    
    events.push({
      id: `evt_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(baseTime + i * 1000 + Math.random() * 500).toISOString(),
      worker,
      model: model.id,
      provider: model.provider,
      promptTokens,
      completionTokens,
      totalTokens,
      cost: Math.round(cost * 10000) / 10000,
      category,
      status,
      pattern: status === "flagged" ? patterns[Math.floor(Math.random() * patterns.length)] : undefined,
    });
  }
  
  return events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
}

function calculateMetrics(events: DemoEvent[]): DemoMetrics {
  const accepted = events.filter(e => e.status === "accepted").length;
  const flagged = events.filter(e => e.status === "flagged").length;
  const totalCost = events.reduce((sum, e) => sum + e.cost, 0);
  const totalTokens = events.reduce((sum, e) => sum + e.totalTokens, 0);
  
  // Cost leaks = sum of flagged event costs
  const costLeaks = events.filter(e => e.status === "flagged").reduce((sum, e) => sum + e.cost, 0);
  
  // Zombie agents = workers with < 20% acceptance rate
  const workerStats: Record<string, { total: number; accepted: number }> = {};
  events.forEach(e => {
    if (!workerStats[e.worker]) workerStats[e.worker] = { total: 0, accepted: 0 };
    workerStats[e.worker].total++;
    if (e.status === "accepted") workerStats[e.worker].accepted++;
  });
  
  const zombieAgents = Object.entries(workerStats)
    .filter(([_, stats]) => stats.total > 3 && stats.accepted / stats.total < 0.2)
    .length;
  
  // Top model by cost
  const modelCosts: Record<string, number> = {};
  events.forEach(e => {
    modelCosts[e.model] = (modelCosts[e.model] || 0) + e.cost;
  });
  const topModel = Object.entries(modelCosts).sort(([,a], [,b]) => b - a)[0]?.[0] || "N/A";
  
  return {
    totalEvents: events.length,
    totalCost,
    totalTokens,
    acceptanceRate: accepted / events.length,
    costLeaks,
    zombieAgents,
    topModel,
    wastePercent: (costLeaks / totalCost) * 100,
  };
}

export function DemoMode() {
  const [isRunning, setIsRunning] = useState(false);
  const [events, setEvents] = useState<DemoEvent[]>([]);
  const [filteredEvents, setFilteredEvents] = useState<DemoEvent[]>([]);
  const [speed, setSpeed] = useState(1);
  const [filterStatus, setFilterStatus] = useState<"all" | "accepted" | "rejected" | "flagged">("all");
  const [intervalId, setIntervalId] = useState<NodeJS.Timeout | null>(null);
  
  const metrics = calculateMetrics(events);

  // Generate initial batch
  useEffect(() => {
    const initial = generateDemoEvents(50);
    setEvents(initial);
    setFilteredEvents(initial);
  }, []);

  // Animation loop
  useEffect(() => {
    if (!isRunning) return;
    
    const interval = setInterval(() => {
      const newEvent = generateDemoEvents(1)[0];
      setEvents(prev => [newEvent, ...prev].slice(0, 200));
    }, 2000 / speed);
    
    setIntervalId(interval);
    return () => { if (intervalId) clearInterval(intervalId); };
  }, [isRunning, speed, intervalId]);

  // Filter events
  useEffect(() => {
    if (filterStatus === "all") {
      setFilteredEvents(events);
    } else {
      setFilteredEvents(events.filter(e => e.status === filterStatus));
    }
  }, [events, filterStatus]);

  const toggleRunning = () => {
    setIsRunning(!isRunning);
  };

  const seedData = () => {
    const fresh = generateDemoEvents(100);
    setEvents(fresh);
    setFilteredEvents(fresh);
  };

  const clearData = () => {
    setEvents([]);
    setFilteredEvents([]);
  };

  return (
    <div className="space-y-6">
      {/* Demo Header */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 p-4 bg-black border border-[#333] rounded-lg"
      >
        <div className="flex items-center gap-4">
          <motion.div
            animate={{ 
              scale: isRunning ? [1, 1.05, 1] : 1,
              backgroundColor: isRunning ? "#22c55e" : "#374151"
            }}
            transition={{ duration: 1, repeat: isRunning ? Infinity : 0 }}
            className="w-3 h-3 rounded-full"
          />
          <div>
            <h3 className="font-bold text-white uppercase tracking-widest">DEMO MODE</h3>
            <p className="text-xs text-zinc-500 font-mono">
              {isRunning ? "STREAMING LIVE" : "PAUSED"} • {speed}x SPEED
            </p>
          </div>
        </div>
        
        <div className="flex flex-wrap items-center gap-3">
          <Button 
            variant={isRunning ? "secondary" : "default"} 
            onClick={toggleRunning}
            className="gap-2"
            size="sm"
          >
            {isRunning ? <Pause size={14} /> : <Play size={14} />}
            {isRunning ? "PAUSE" : "START"}
          </Button>
          
          <Button variant="ghost" size="sm" onClick={seedData}>
            <RefreshCw size={14} className="mr-1" />
            RESEED
          </Button>
          
          <Button variant="ghost" size="sm" onClick={clearData}>
            <X size={14} className="mr-1" />
            CLEAR
          </Button>

          <select 
            aria-label="Simulation Speed"
            title="Simulation Speed"
            value={speed} 
            onChange={(e) => setSpeed(Number(e.target.value))}
            className="bg-[#111] border border-[#333] text-white text-xs px-2 py-1 rounded"
          >
            <option value={1}>1x</option>
            <option value={2}>2x</option>
            <option value={5}>5x</option>
            <option value={10}>10x</option>
          </select>

          <select 
            aria-label="Event Filter"
            title="Event Filter"
            value={filterStatus} 
            onChange={(e) => setFilterStatus(e.target.value as any)}
            className="bg-[#111] border border-[#333] text-white text-xs px-2 py-1 rounded"
          >
            <option value="all">ALL</option>
            <option value="accepted">ACCEPTED</option>
            <option value="rejected">REJECTED</option>
            <option value="flagged">FLAGGED</option>
          </select>
        </div>
      </motion.div>

      {/* Metrics Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-4">
        <MetricCard 
          title="TOTAL EVENTS" 
          value={metrics.totalEvents.toLocaleString()} 
          icon={Terminal} 
          color="#ffb000"
          trend="+12%"
        />
        <MetricCard 
          title="TOTAL SPEND" 
          value={`$${formatMoney(metrics.totalCost)}`} 
          icon={DollarSign} 
          color="#22c55e"
          trend="-3.2%"
        />
        <MetricCard 
          title="ACCEPTANCE" 
          value={`${(metrics.acceptanceRate * 100).toFixed(1)}%`} 
          icon={Check} 
          color="#06b6d4"
          trend="+1.4%"
        />
        <MetricCard 
          title="WASTE %" 
          value={`${metrics.wastePercent.toFixed(1)}%`} 
          icon={AlertTriangle} 
          color="#ef4444"
          trend="-0.8%"
        />
        <MetricCard 
          title="COST LEAKS" 
          value={`$${formatMoney(metrics.costLeaks)}`} 
          icon={Zap} 
          color="#f97316"
        />
        <MetricCard 
          title="ZOMBIE AGENTS" 
          value={metrics.zombieAgents.toString()} 
          icon={Brain} 
          color="#8b5cf6"
        />
        <MetricCard 
          title="TOP MODEL" 
          value={metrics.topModel} 
          icon={TrendingUp} 
          color="#a855f7"
        />
        <MetricCard 
          title="TOTAL TOKENS" 
          value={metrics.totalTokens.toLocaleString()} 
          icon={Sparkles} 
          color="#84cc16"
        />
      </div>

      {/* Live Event Stream */}
      <Card className="bg-black border-[#333]">
        <CardHeader className="border-b border-[#333] bg-[#0a0a0a] flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
              <Terminal size={18} className="#ffb000" />
              LIVE EVENT STREAM
            </CardTitle>
            <p className="text-xs text-zinc-500 font-mono">
              {filteredEvents.length} events • {events.length} total
            </p>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="max-h-[500px] overflow-y-auto">
            <table className="w-full text-xs">
              <thead className="bg-[#111] sticky top-0 border-b border-[#333]">
                <tr>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">TIME</th>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">WORKER</th>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">MODEL</th>
                  <th className="px-4 py-3 text-left text-zinc-500 uppercase tracking-widest font-normal">CATEGORY</th>
                  <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">TOKENS</th>
                  <th className="px-4 py-3 text-right text-zinc-500 uppercase tracking-widest font-normal">COST</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">STATUS</th>
                  <th className="px-4 py-3 text-center text-zinc-500 uppercase tracking-widest font-normal">PATTERN</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#222]">
                <AnimatePresence mode="popLayout">
                  {filteredEvents.slice(0, 50).map((event, index) => (
                    <motion.tr
                      key={event.id}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: 20 }}
                      transition={{ delay: index * 0.02 }}
                      className={`hover:bg-[#0a0a0a] transition-colors ${event.status === "flagged" ? "bg-red-900/20" : ""}`}
                    >
                      <td className="px-4 py-3 text-zinc-400 font-mono">
                        {new Date(event.timestamp).toLocaleTimeString()}
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-zinc-300 font-mono text-sm">{event.worker}</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-zinc-400 font-mono text-sm">{event.model}</span>
                        <span className="text-[10px] text-zinc-600 ml-1">({event.provider})</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-0.5 text-[10px] rounded font-mono ${
                          event.category === "code_generation" ? "bg-blue-900/30 text-blue-400" :
                          event.category === "code_review" ? "bg-green-900/30 text-green-400" :
                          event.category === "testing" ? "bg-yellow-900/30 text-yellow-400" :
                          "bg-purple-900/30 text-purple-400"
                        }`}>
                          {event.category.toUpperCase()}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right text-zinc-400 font-mono">
                        {event.totalTokens.toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <span className={`font-mono font-bold ${event.cost > 0.50 ? "text-red-400" : "text-zinc-300"}`}>
                          ${formatMoney(event.cost)}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span className={`px-2 py-1 text-[10px] rounded font-bold uppercase tracking-wider ${
                          event.status === "accepted" ? "bg-green-900/30 text-green-400" :
                          event.status === "rejected" ? "bg-red-900/30 text-red-400" :
                          "bg-yellow-900/30 text-yellow-400"
                        }`}>
                          {event.status.toUpperCase()}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        {event.pattern ? (
                          <span className="px-2 py-1 text-[10px] bg-red-900/30 text-red-400 rounded font-mono uppercase">
                            {event.pattern.toUpperCase()}
                          </span>
                        ) : (
                          <span className="text-zinc-600 text-[10px]">—</span>
                        )}
                      </td>
                    </motion.tr>
                  ))}
                  {filteredEvents.length === 0 && (
                    <tr>
                      <td colSpan={8} className="px-4 py-12 text-center text-zinc-600 uppercase tracking-widest">
                        NO EVENTS MATCH FILTER
                      </td>
                    </tr>
                  )}
                </AnimatePresence>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Cost Attribution Demo */}
      <motion.section
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="space-y-6"
      >
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-bold text-white uppercase tracking-widest flex items-center gap-2">
            <ArrowRight size={18} className="#ffb000" />
            COST ATTRIBUTION DEMO
          </h3>
          <Button variant="outline" size="sm">
            <Download size={14} className="mr-1" />
            EXPORT CSV
          </Button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* By Worker */}
          <Card className="bg-black border-[#333]">
            <CardHeader>
              <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
                <Terminal size={16} className="#ffb000" />
                BY WORKER
              </CardTitle>
            </CardHeader>
            <CardContent>
              <WorkerCostTable events={events} />
            </CardContent>
          </Card>

          {/* By Model */}
          <Card className="bg-black border-[#333]">
            <CardHeader>
              <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
                <Brain size={16} className="#06b6d4" />
                BY MODEL
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ModelCostTable events={events} />
            </CardContent>
          </Card>

          {/* By Category */}
          <Card className="bg-black border-[#333]">
            <CardHeader>
              <CardTitle className="text-white font-bold uppercase tracking-widest flex items-center gap-2">
                <Zap size={16} className="#ef4444" />
                BY CATEGORY
              </CardTitle>
            </CardHeader>
            <CardContent>
              <CategoryCostTable events={events} />
            </CardContent>
          </Card>
        </div>
      </motion.section>
    </div>
  );
}

function MetricCard({ 
  title, 
  value, 
  icon: Icon, 
  color, 
  trend 
}: { 
  title: string; 
  value: string; 
  icon: React.ComponentType<any>;
  color: string;
  trend?: string;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="p-4 bg-black border border-[#333] rounded-lg"
    >
      <div className="flex items-center justify-between mb-2">
        <Icon size={16} style={{ color }} />
        {trend && (
          <span className={`text-[10px] font-bold uppercase tracking-widest ${trend.startsWith("-") ? "text-green-400" : "text-red-400"}`}>
            {trend}
          </span>
        )}
      </div>
      <p className="text-zinc-500 text-[10px] uppercase tracking-widest mb-1">{title}</p>
      <p className="text-white font-bold text-xl font-mono">{value}</p>
    </motion.div>
  );
}

function WorkerCostTable({ events }: { events: DemoEvent[] }) {
  const workerStats: Record<string, { cost: number; events: number; tokens: number }> = {};
  
  events.forEach(e => {
    if (!workerStats[e.worker]) workerStats[e.worker] = { cost: 0, events: 0, tokens: 0 };
    workerStats[e.worker].cost += e.cost;
    workerStats[e.worker].events += 1;
    workerStats[e.worker].tokens += e.totalTokens;
  });

  return (
    <div className="space-y-2 max-h-64 overflow-y-auto">
      {Object.entries(workerStats)
        .sort(([,a], [,b]) => b.cost - a.cost)
        .slice(0, 8)
        .map(([worker, stats], i) => (
          <div key={worker} className="flex justify-between items-center p-2 bg-[#111] rounded hover:bg-[#1a1a1a] transition-colors">
            <div className="flex items-center gap-3">
              <span className="w-6 h-6 rounded bg-[#ffb000]20 flex items-center justify-center text-[#ffb000] text-[10px] font-bold">
                {i + 1}
              </span>
              <span className="text-zinc-300 font-mono text-sm">{worker}</span>
            </div>
            <div className="text-right">
              <p className="text-white font-bold text-sm">${formatMoney(stats.cost)}</p>
              <p className="text-[10px] text-zinc-500">{stats.events} events • {stats.tokens.toLocaleString()} tok</p>
            </div>
          </div>
        ))}
      {Object.keys(workerStats).length === 0 && (
        <p className="text-zinc-600 text-center py-8 text-xs uppercase">NO DATA</p>
      )}
    </div>
  );
}

function ModelCostTable({ events }: { events: DemoEvent[] }) {
  const modelStats: Record<string, { cost: number; events: number; provider: string }> = {};
  
  events.forEach(e => {
    if (!modelStats[e.model]) modelStats[e.model] = { cost: 0, events: 0, provider: e.provider };
    modelStats[e.model].cost += e.cost;
    modelStats[e.model].events += 1;
  });

  return (
    <div className="space-y-2 max-h-64 overflow-y-auto">
      {Object.entries(modelStats)
        .sort(([,a], [,b]) => b.cost - a.cost)
        .slice(0, 8)
        .map(([model, stats], i) => (
          <div key={model} className="flex justify-between items-center p-2 bg-[#111] rounded hover:bg-[#1a1a1a] transition-colors">
            <div className="flex items-center gap-3">
              <span className="w-6 h-6 rounded bg-[#06b6d4]20 flex items-center justify-center text-[#06b6d4] text-[10px] font-bold">
                {i + 1}
              </span>
              <span className="text-zinc-300 font-mono text-sm">{model}</span>
              <span className="text-[10px] text-zinc-500">{stats.provider}</span>
            </div>
            <div className="text-right">
              <p className="text-white font-bold text-sm">${formatMoney(stats.cost)}</p>
              <p className="text-[10px] text-zinc-500">{stats.events} events</p>
            </div>
          </div>
        ))}
      {Object.keys(modelStats).length === 0 && (
        <p className="text-zinc-600 text-center py-8 text-xs uppercase">NO DATA</p>
      )}
    </div>
  );
}

function CategoryCostTable({ events }: { events: DemoEvent[] }) {
  const catStats: Record<string, { cost: number; events: number; accepted: number }> = {};
  
  events.forEach(e => {
    if (!catStats[e.category]) catStats[e.category] = { cost: 0, events: 0, accepted: 0 };
    catStats[e.category].cost += e.cost;
    catStats[e.category].events += 1;
    if (e.status === "accepted") catStats[e.category].accepted += 1;
  });

  const colors: Record<string, string> = {
    code_generation: "#3b82f6",
    code_review: "#22c55e",
    testing: "#eab308",
    documentation: "#a855f7",
    debugging: "#ef4444",
    architecture: "#06b6d4",
    refactoring: "#f97316",
    security_audit: "#ec4899",
  };

  return (
    <div className="space-y-2 max-h-64 overflow-y-auto">
      {Object.entries(catStats)
        .sort(([,a], [,b]) => b.cost - a.cost)
        .map(([cat, stats]) => {
          const color = colors[cat] || "#888";
          const acceptanceRate = stats.events > 0 ? (stats.accepted / stats.events) * 100 : 0;
          return (
            <div key={cat} className="flex justify-between items-center p-2 bg-[#111] rounded hover:bg-[#1a1a1a] transition-colors">
              <div className="flex items-center gap-3">
                <span className="w-3 h-3 rounded" style={{ backgroundColor: color }} />
                <span className="text-zinc-300 font-mono text-sm capitalize">{cat.replace("_", " ")}</span>
              </div>
              <div className="text-right">
                <p className="text-white font-bold text-sm">${formatMoney(stats.cost)}</p>
                <p className="text-[10px] text-zinc-500">{stats.events} events • {acceptanceRate.toFixed(0)}% accept</p>
              </div>
            </div>
          );
        })}
      {Object.keys(catStats).length === 0 && (
        <p className="text-zinc-600 text-center py-8 text-xs uppercase">NO DATA</p>
      )}
    </div>
  );
}


function Button({ children, variant, size, className, ...props }: any) {
  return <button className={`px-4 py-2 text-xs font-bold uppercase tracking-widest rounded transition-colors ${variant === 'ghost' ? 'hover:bg-[#111] text-zinc-400' : variant === 'outline' ? 'border border-[#333] hover:border-zinc-500' : 'bg-[#ffb000] text-black hover:bg-[#ff8c00]'} ${className}`} {...props}>{children}</button>
}
function Card({ children, className }: any) {
  return <div className={`border rounded ${className}`}>{children}</div>
}
function CardHeader({ children, className }: any) {
  return <div className={`p-4 border-b ${className}`}>{children}</div>
}
function CardTitle({ children, className }: any) {
  return <h3 className={`font-bold ${className}`}>{children}</h3>
}
function CardContent({ children, className }: any) {
  return <div className={`p-4 ${className}`}>{children}</div>
}