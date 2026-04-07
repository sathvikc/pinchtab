import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAppStore } from "../stores/useAppStore";
import { EmptyState, ErrorBoundary } from "../components/atoms";
import InstanceListItem from "../instances/InstanceListItem";
import InstanceTabsPanel from "../tabs/InstanceTabsPanel";
import * as api from "../services/api";

export default function MonitoringPage() {
  const {
    instances,
    currentTabs,
    currentMemory,
    settings,
    monitoringSidebarCollapsed: sidebarCollapsed,
    setMonitoringSidebarCollapsed: setSidebarCollapsed,
    selectedMonitoringInstanceId,
    setSelectedMonitoringInstanceId,
  } = useAppStore();
  const navigate = useNavigate();
  const selectedId = selectedMonitoringInstanceId;
  const setSelectedId = setSelectedMonitoringInstanceId;
  const [strategy, setStrategy] = useState<string>("always-on");
  const memoryEnabled = settings.monitoring?.memoryMetrics ?? false;

  // Fetch backend strategy once
  useEffect(() => {
    const load = async () => {
      try {
        const cfg = await api.fetchBackendConfig();
        setStrategy(cfg.config.multiInstance.strategy);
      } catch {
        // ignore — default to always-on
      }
    };
    load();
  }, []);

  // Auto-select first running instance
  useEffect(() => {
    if (!selectedId) {
      const firstRunning = instances.find((i) => i.status === "running");
      if (firstRunning) setSelectedId(firstRunning.id);
    }
  }, [instances, selectedId, setSelectedId]);

  const handleStop = async (id: string) => {
    try {
      await api.stopInstance(id);
    } catch (e) {
      console.error("Failed to stop instance", e);
    }
  };

  const selectedInstance = instances?.find((i) => i.id === selectedId);
  const selectedTabs = selectedId ? currentTabs?.[selectedId] || [] : [];

  return (
    <ErrorBoundary>
      <div className="flex h-full flex-col overflow-hidden">
        {instances.length === 0 && (
          <div className="flex flex-1 items-center justify-center">
            <EmptyState
              title="No instances yet"
              description="Start a profile to see instance data"
              icon="📡"
            />
          </div>
        )}

        {instances.length > 0 && (
          <div className="dashboard-panel flex flex-1 flex-col overflow-hidden rounded-none!">
            <div className="flex flex-1 overflow-hidden">
              {!sidebarCollapsed && (
                <div className="w-64 shrink-0 overflow-auto border-r border-border-subtle bg-bg-surface/50">
                  <div className="flex items-center justify-between border-b border-border-subtle px-3 py-1.5">
                    <span className="text-xs font-medium text-text-muted">
                      Instances
                    </span>
                    <button
                      type="button"
                      onClick={() => setSidebarCollapsed(true)}
                      title="Collapse sidebar"
                      className="rounded p-1 text-text-muted transition-colors hover:bg-white/10 hover:text-text-secondary"
                    >
                      <svg
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                        className="h-3.5 w-3.5"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <polyline points="15 18 9 12 15 6" />
                      </svg>
                    </button>
                  </div>
                  <div>
                    {instances.map((inst) => (
                      <InstanceListItem
                        key={inst.id}
                        instance={inst}
                        tabCount={currentTabs[inst.id]?.length ?? 0}
                        memoryMB={
                          memoryEnabled ? currentMemory[inst.id] : undefined
                        }
                        selected={selectedId === inst.id}
                        autoRestart={
                          inst.profileName === "default" &&
                          (strategy === "always-on" ||
                            strategy === "simple-autorestart")
                        }
                        onClick={() => setSelectedId(inst.id)}
                        onStop={() => handleStop(inst.id)}
                        onOpenProfile={() =>
                          navigate("/dashboard/profiles", {
                            state: {
                              selectedProfileKey:
                                inst.profileId || inst.profileName,
                            },
                          })
                        }
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Selected instance details */}
              <div className="flex flex-1 flex-col overflow-hidden">
                {selectedInstance ? (
                  <InstanceTabsPanel
                    tabs={selectedTabs}
                    instanceId={selectedId || undefined}
                  />
                ) : (
                  <div className="flex flex-1 items-center justify-center text-sm text-text-muted">
                    Select an instance to view details
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </ErrorBoundary>
  );
}
