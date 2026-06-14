"use client";

import { useState, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  Zap, 
  Shield, 
  BarChart2, 
  Target, 
  ArrowRight, 
  X, 
  Check, 
  ChevronRight,
  ChevronLeft,
  Sparkles,
  Terminal,
  Brain,
  DollarSign
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const TOUR_STEPS = [
  {
    id: "welcome",
    title: "Welcome to TokenGoblin",
    description: "Your autonomous AI spend observability system. Track every token, optimize every call, eliminate waste automatically.",
    icon: Sparkles,
    color: "#ffb000",
    highlight: null,
  },
  {
    id: "scorecard",
    title: "Executive Scorecard",
    description: "Real-time AI maturity grade, projected spend, fleet ROI, and system reliability — all in one view.",
    icon: BarChart2,
    color: "#ffb000",
    highlight: "[data-tour=\"scorecard\"]",
  },
  {
    id: "cost-leaks",
    title: "Cost Leak Detection",
    description: "Automatically identify waste patterns: over-tokening, hallucination loops, redundant calls, and zombie agents.",
    icon: Shield,
    color: "#ef4444",
    highlight: "[data-tour=\"cost-leaks\"]",
  },
  {
    id: "zombie-agents",
    title: "Zombie Agent Forensics",
    description: "Detect agents with near-zero acceptance rates that drain budget without producing value.",
    icon: Brain,
    color: "#f97316",
    highlight: "[data-tour=\"zombie-agents\"]",
  },
  {
    id: "prompt-graveyard",
    title: "Prompt Graveyard",
    description: "Forensic analysis of dead prompts — total waste, acceptance rates, and exact fingerprints for remediation.",
    icon: Target,
    color: "#8b5cf6",
    highlight: "[data-tour=\"prompt-graveyard\"]",
  },
  {
    id: "model-matrix",
    title: "Model Performance Matrix",
    description: "Compare every model by cost/call, cost/outcome, quality score, and latency. Route smarter, spend less.",
    icon: BarChart2,
    color: "#06b6d4",
    highlight: "[data-tour=\"model-matrix\"]",
  },
  {
    id: "forecast",
    title: "Spend Forecasting",
    description: "ML-powered projections with confidence intervals. Never be surprised by your AI bill again.",
    icon: DollarSign,
    color: "#22c55e",
    highlight: "[data-tour=\"forecast\"]",
  },
  {
    id: "actions",
    title: "Take Action",
    description: "Seed demo data, sync live, export reports, manage API keys. Everything you need to operate.",
    icon: Terminal,
    color: "#ffb000",
    highlight: "[data-tour=\"actions\"]",
  },
];

interface OnboardingTourProps {
  isOpen: boolean;
  onClose: () => void;
  onComplete: () => void;
}

export function OnboardingTour({ isOpen, onClose, onComplete }: OnboardingTourProps) {
  const [currentStep, setCurrentStep] = useState(0);
  const [completed, setCompleted] = useState(false);

  const totalSteps = TOUR_STEPS.length;

  if (!isOpen) return null;

  const step = TOUR_STEPS[currentStep];

  const goNext = useCallback(() => {
    if (currentStep < totalSteps - 1) {
      setCurrentStep(currentStep + 1);
    } else {
      setCompleted(true);
      setTimeout(() => {
        onComplete();
        onClose();
      }, 300);
    }
  }, [currentStep, totalSteps, onComplete, onClose]);

  const goPrev = useCallback(() => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  }, [currentStep]);

  const skipTour = useCallback(() => {
    onClose();
  }, [onClose]);

  return (
    <AnimatePresence mode="wait">
      {isOpen && (
        <>
          {/* Tour Modal */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm"
            onClick={skipTour}
            aria-hidden="true"
          />
          
          {/* Tour Content */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: -20 }}
            transition={{ duration: 0.3, type: "spring", damping: 25, stiffness: 300 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="w-full max-w-2xl relative pointer-events-auto">
              {/* Progress Bar */}
              <motion.div
                className="w-full h-1 bg-[#111] rounded-full overflow-hidden mb-6"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.2 }}
              >
                <motion.div
                  className="h-full bg-gradient-to-r from-[#ffb000] to-[#ff8c00] rounded-full"
                  initial={{ width: 0 }}
                  animate={{ width: `${((currentStep + 1) / totalSteps) * 100}%` }}
                  transition={{ duration: 0.4, ease: "easeOut" }}
                />
              </motion.div>

              {/* Step Counter */}
              <div className="flex justify-between items-center mb-6 text-xs font-bold uppercase tracking-widest text-zinc-500">
                <span>STEP {currentStep + 1} / {totalSteps}</span>
                <span>{Math.round(((currentStep + 1) / totalSteps) * 100)}% COMPLETE</span>
              </div>

              {/* Tour Card */}
              <Card className="bg-black border-[#333] overflow-hidden">
                <CardHeader className="pb-4">
                  <div className="flex items-center gap-4">
                    <motion.div
                      initial={{ scale: 0, rotate: -45 }}
                      animate={{ scale: 1, rotate: 0 }}
                      transition={{ type: "spring", damping: 15, stiffness: 200 }}
                      className={`w-14 h-14 rounded-xl flex items-center justify-center flex-shrink-0 ${step.color}20`}
                    >
                      <step.icon size={28} className={step.color} />
                    </motion.div>
                    <div>
                      <h3 className="text-2xl font-bold text-white tracking-widest">{step.title}</h3>
                      <p className="text-zinc-400 text-sm mt-1 font-mono">{step.description}</p>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <div className="space-y-4">
                    {/* Feature Highlights */}
                    <div className="grid gap-3 sm:grid-cols-2">
                      <FeatureHighlight icon={Zap} title="Real-time" desc="Live telemetry streaming" />
                      <FeatureHighlight icon={Shield} title="Automated" desc="Zero-config detection" />
                    </div>

                    {/* Action Button */}
                    <motion.button
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                      onClick={currentStep === totalSteps - 1 ? goNext : goNext}
                      className="w-full py-3 px-4 rounded-lg font-bold uppercase tracking-widest text-sm transition-all flex items-center justify-center gap-2"
                      style={{
                        backgroundColor: currentStep === totalSteps - 1 ? "#ffb000" : "#111",
                        color: currentStep === totalSteps - 1 ? "black" : "#fff",
                        border: currentStep === totalSteps - 1 ? "none" : "1px solid #333",
                      }}
                    >
                      {currentStep === totalSteps - 1 
                        ? "ENTER COMMAND CENTER" 
                        : "CONTINUE"}
                      <ChevronRight size={16} />
                    </motion.button>

                    {/* Skip/Back */}
                    <div className="flex justify-between pt-2">
                      {currentStep > 0 && (
                        <Button variant="ghost" size="sm" onClick={goPrev} className="text-zinc-500 hover:text-white">
                          <ChevronLeft size={14} className="mr-1" />
                          BACK
                        </Button>
                      )}
                      <Button variant="ghost" size="sm" onClick={skipTour} className="text-zinc-500 hover:text-white">
                        SKIP TOUR
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Completion State */}
              {completed && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="mt-4 p-6 bg-[#ffb000]20 border border-[#ffb000]40 rounded-xl text-center"
                >
                  <div className="w-16 h-16 mx-auto mb-4 bg-[#ffb000] rounded-full flex items-center justify-center">
                    <Check size={32} className="text-black" />
                  </div>
                  <h4 className="text-lg font-bold text-[#ffb000] uppercase tracking-widest mb-2">ONBOARDING COMPLETE</h4>
                  <p className="text-zinc-400 text-sm">You're ready to start optimizing your AI spend.</p>
                  <motion.button
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    onClick={onClose}
                    className="mt-4 bg-[#ffb000] text-black px-6 py-2 rounded font-bold uppercase tracking-widest text-sm hover:bg-[#ff8c00] transition-colors"
                  >
                    LAUNCH DASHBOARD
                  </motion.button>
                </motion.div>
              )}
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}

function FeatureHighlight({ icon: Icon, title, desc }: { icon: React.ComponentType<{ size?: number }>, title: string, desc: string }) {
  return (
    <div className="p-4 bg-[#111] border border-[#222] rounded-lg group hover:border-[#333] transition-colors">
      <Icon size={20} className="#ffb000 mb-2" />
      <p className="text-xs font-bold uppercase tracking-widest text-white mb-1">{title}</p>
      <p className="text-[10px] text-zinc-500 uppercase tracking-widest">{desc}</p>
    </div>
  );
}