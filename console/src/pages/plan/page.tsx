import type React from "react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  forecastActionErrorMessage,
  isForecastMissing,
  useCreateForecast,
  useForecastScenarios,
  useLatestForecast,
  useRunScenario,
  type Forecast,
  type Scenario,
} from "@/hooks/use-forecast";
import {
  decisionActionErrorMessage,
  dueInDays,
  useCreateDecision,
  useOpenActions,
  useUpdateActionStatus,
  type Action,
} from "@/hooks/use-decisions";
import { cn } from "@/lib/utils";

const PlanPage: React.FC = () => {
  const query = useLatestForecast();
  const create = useCreateForecast();
  const [actionError, setActionError] = useState<string | null>(null);
  const [startingBalance, setStartingBalance] = useState("2000.00");
  const [disabledIds, setDisabledIds] = useState<string[]>([]);
  const [includeVariable, setIncludeVariable] = useState(true);
  const [includeUncertain, setIncludeUncertain] = useState(true);
  const [tab, setTab] = useState("forecast");

  const forecast = query.data;
  const missing = !forecast && query.isError && isForecastMissing(query.error);
  const busy = create.isPending;
  const hasForecast = Boolean(forecast);

  const dirty = useMemo(() => {
    if (!forecast) {
      return false;
    }
    const savedDisabled = [...(forecast.assumptions.disabled_series_ids ?? [])].sort();
    const localDisabled = [...disabledIds].sort();
    return (
      normalizeAmount(startingBalance) !== forecast.starting_balance ||
      includeVariable !== forecast.assumptions.include_variable ||
      includeUncertain !== forecast.assumptions.include_uncertain ||
      savedDisabled.join("|") !== localDisabled.join("|")
    );
  }, [forecast, startingBalance, disabledIds, includeVariable, includeUncertain]);

  useEffect(() => {
    if (!forecast) {
      return;
    }
    setStartingBalance(forecast.starting_balance);
    setDisabledIds(forecast.assumptions.disabled_series_ids ?? []);
    setIncludeVariable(forecast.assumptions.include_variable);
    setIncludeUncertain(forecast.assumptions.include_uncertain);
  }, [forecast?.id]);

  const onUpdateForecast = async () => {
    setActionError(null);
    try {
      const result = await create.mutateAsync({
        body: {
          starting_balance: normalizeAmount(startingBalance),
          horizon_days: 90,
          assumptions: {
            disabled_series_ids: disabledIds,
            include_variable: includeVariable,
            include_uncertain: includeUncertain,
          },
        },
      });
      setDisabledIds(result.assumptions.disabled_series_ids);
      setIncludeVariable(result.assumptions.include_variable);
      setIncludeUncertain(result.assumptions.include_uncertain);
    } catch (err) {
      setActionError(forecastActionErrorMessage(err));
    }
  };

  const toggleSeries = (id: string, enabled: boolean) => {
    setDisabledIds((prev) => {
      if (enabled) {
        return prev.filter((x) => x !== id);
      }
      return prev.includes(id) ? prev : [...prev, id];
    });
  };

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 pb-10">
        {actionError ? (
          <p className="text-sm text-destructive" role="alert">
            {actionError}
          </p>
        ) : null}

        {query.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : query.isError && !missing ? (
          <p className="text-sm text-destructive" role="alert">
            Could not load the forecast.
          </p>
        ) : !hasForecast ? (
          <SetupForecast
            startingBalance={startingBalance}
            onStartingBalanceChange={setStartingBalance}
            busy={busy}
            onCreate={onUpdateForecast}
          />
        ) : (
          <Tabs
            value={tab}
            onValueChange={(value) => {
              if (typeof value === "string") {
                setTab(value);
              }
            }}
            className="gap-6"
          >
            <TabsList variant="line" className="w-full justify-start">
              <TabsTrigger value="forecast">Forecast</TabsTrigger>
              <TabsTrigger value="what-if">What if</TabsTrigger>
              <TabsTrigger value="actions">Actions</TabsTrigger>
            </TabsList>

            <TabsContent value="forecast" className="flex flex-col gap-8">
              <ForecastSummary forecast={forecast!} />
              <AssumptionsPanel
                startingBalance={startingBalance}
                onStartingBalanceChange={setStartingBalance}
                includeVariable={includeVariable}
                onIncludeVariableChange={setIncludeVariable}
                includeUncertain={includeUncertain}
                onIncludeUncertainChange={setIncludeUncertain}
                seriesOptions={forecast!.series_options}
                disabledIds={disabledIds}
                onToggleSeries={toggleSeries}
                dirty={dirty}
                busy={busy}
                onUpdate={onUpdateForecast}
              />
            </TabsContent>

            <TabsContent value="what-if" className="flex flex-col gap-5">
              {dirty ? (
                <div className="flex flex-col gap-3 rounded-lg border bg-muted/40 px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-sm text-muted-foreground">
                    Forecast assumptions changed. Update the Forecast tab
                    before comparing a what-if.
                  </p>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setTab("forecast")}
                  >
                    Go to Forecast
                  </Button>
                </div>
              ) : null}
              <ScenarioPanel
                forecastId={forecast!.id}
                assumptionsDirty={dirty}
              />
            </TabsContent>

            <TabsContent value="actions" className="flex flex-col gap-5">
              <OpenActionsPanel />
            </TabsContent>
          </Tabs>
        )}
      </div>
    </div>
  );
};

type SetupForecastProps = {
  startingBalance: string;
  onStartingBalanceChange: (value: string) => void;
  busy: boolean;
  onCreate: () => void;
};

const SetupForecast: React.FC<SetupForecastProps> = ({
  startingBalance,
  onStartingBalanceChange,
  busy,
  onCreate,
}) => {
  return (
    <section className="flex flex-col gap-5">
      <div className="rounded-xl border border-dashed px-4 py-10 text-center">
        <p className="text-sm font-medium">No forecast yet</p>
        <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
          Create a 90-day forecast first. What-if comparisons become available
          after that.
        </p>
      </div>
      <label className="mx-auto flex w-full max-w-xs flex-col gap-1.5 text-xs text-muted-foreground">
        Starting balance (EUR)
        <input
          className="h-10 rounded-lg border border-input bg-background px-3 text-sm tabular-nums text-foreground"
          inputMode="decimal"
          value={startingBalance}
          disabled={busy}
          onChange={(e) => onStartingBalanceChange(e.target.value)}
        />
      </label>
      <div className="flex justify-center">
        <Button type="button" disabled={busy} onClick={onCreate}>
          {busy ? "Calculating…" : "Show 90-day forecast"}
        </Button>
      </div>
    </section>
  );
};

type AssumptionsPanelProps = {
  startingBalance: string;
  onStartingBalanceChange: (value: string) => void;
  includeVariable: boolean;
  onIncludeVariableChange: (value: boolean) => void;
  includeUncertain: boolean;
  onIncludeUncertainChange: (value: boolean) => void;
  seriesOptions: Forecast["series_options"];
  disabledIds: string[];
  onToggleSeries: (id: string, enabled: boolean) => void;
  dirty: boolean;
  busy: boolean;
  onUpdate: () => void;
};

const AssumptionsPanel: React.FC<AssumptionsPanelProps> = ({
  startingBalance,
  onStartingBalanceChange,
  includeVariable,
  onIncludeVariableChange,
  includeUncertain,
  onIncludeUncertainChange,
  seriesOptions,
  disabledIds,
  onToggleSeries,
  dirty,
  busy,
  onUpdate,
}) => {
  return (
    <section className="flex flex-col gap-4">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight">Assumptions</h2>
        <p className="text-sm text-muted-foreground">
          Only change these when your starting balance or recurring picture
          changes.
        </p>
      </div>

      <label className="flex max-w-xs flex-col gap-1.5 text-xs text-muted-foreground">
        Starting balance (EUR)
        <input
          className="h-10 rounded-lg border border-input bg-background px-3 text-sm tabular-nums text-foreground"
          inputMode="decimal"
          value={startingBalance}
          disabled={busy}
          onChange={(e) => onStartingBalanceChange(e.target.value)}
        />
      </label>

      <label className="flex items-center gap-2.5 text-sm text-muted-foreground">
        <input
          type="checkbox"
          className="size-3.5 accent-foreground"
          checked={includeVariable}
          disabled={busy}
          onChange={(e) => onIncludeVariableChange(e.target.checked)}
        />
        Include average variable spend
      </label>
      <label className="flex items-center gap-2.5 text-sm text-muted-foreground">
        <input
          type="checkbox"
          className="size-3.5 accent-foreground"
          checked={includeUncertain}
          disabled={busy}
          onChange={(e) => onIncludeUncertainChange(e.target.checked)}
        />
        Include uncertain recurring series
      </label>

      {seriesOptions?.length ? (
        <ul className="divide-y border-y">
          {seriesOptions.map((opt) => {
            const enabled = !disabledIds.includes(opt.id);
            return (
              <li
                key={opt.id}
                className="flex items-center justify-between gap-3 py-3"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{opt.display_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {opt.interval} · {formatAmount(opt.amount)}
                  </p>
                </div>
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    className="size-3.5 accent-foreground"
                    checked={enabled}
                    disabled={busy}
                    onChange={(e) => onToggleSeries(opt.id, e.target.checked)}
                  />
                  On
                </label>
              </li>
            );
          })}
        </ul>
      ) : null}

      {dirty ? (
        <div className="flex items-center justify-between gap-3 rounded-lg border bg-muted/40 px-3 py-2.5">
          <p className="text-sm text-muted-foreground">
            Assumptions changed — update to apply them to the forecast.
          </p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={onUpdate}
          >
            {busy ? "Updating…" : "Update forecast"}
          </Button>
        </div>
      ) : null}
    </section>
  );
};

const ForecastSummary: React.FC<{ forecast: Forecast }> = ({ forecast }) => {
  const minBalance = Number.parseFloat(forecast.min_balance);
  const endingBalance = Number.parseFloat(forecast.ending_balance);
  const goesNegative = minBalance < 0;
  const horizonDays = forecast.horizon_days || 90;

  return (
    <section className="flex flex-col gap-5">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight">
          Next {horizonDays} days
        </h2>
        <p className="text-sm text-muted-foreground">
          Projected cash balance if your current recurring picture continues.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Stat
          label="Today"
          hint="Starting balance"
          value={formatAmount(forecast.starting_balance)}
        />
        <Stat
          label="Lowest point"
          hint="Smallest balance in this period"
          value={formatAmount(forecast.min_balance)}
          className={
            goesNegative
              ? "text-red-700 dark:text-red-400"
              : "text-foreground"
          }
        />
        <Stat
          label={`After ${horizonDays} days`}
          hint="Balance at the end"
          value={formatAmount(forecast.ending_balance)}
          className={
            endingBalance < 0
              ? "text-red-700 dark:text-red-400"
              : "text-foreground"
          }
        />
      </div>

      {goesNegative ? (
        <p className="text-sm text-red-700 dark:text-red-400">
          At some point in the next {horizonDays} days your balance would drop
          below zero — you would run out of cash unless something changes.
        </p>
      ) : null}

      <details className="group">
        <summary className="cursor-pointer list-none text-sm text-muted-foreground underline-offset-4 hover:underline [&::-webkit-details-marker]:hidden">
          Week by week
        </summary>
        <ul className="mt-3 divide-y border-y">
          {forecast.points.map((p) => (
            <li
              key={p.date}
              className="flex items-center justify-between py-2 text-sm"
            >
              <span className="text-muted-foreground">{formatDate(p.date)}</span>
              <span className="font-medium tabular-nums">
                {formatAmount(p.balance)}
              </span>
            </li>
          ))}
        </ul>
      </details>
    </section>
  );
};

const ScenarioPanel: React.FC<{
  forecastId: string;
  assumptionsDirty: boolean;
}> = ({ forecastId, assumptionsDirty }) => {
  const scenariosQuery = useForecastScenarios(forecastId);
  const run = useRunScenario(forecastId);
  const [kind, setKind] = useState<
    "new_monthly_obligation" | "income_gap" | "one_off_plus_goal"
  >("new_monthly_obligation");
  const [monthlyAmount, setMonthlyAmount] = useState("100.00");
  const [startDate, setStartDate] = useState(todayISO());
  const [incomeDelta, setIncomeDelta] = useState("-500.00");
  const [months, setMonths] = useState("3");
  const [oneOff, setOneOff] = useState("1000.00");
  const [goal, setGoal] = useState("2000.00");
  const [byDate, setByDate] = useState(todayISO());
  const [error, setError] = useState<string | null>(null);

  const scenarios = scenariosQuery.data?.data ?? [];

  const onRun = async () => {
    setError(null);
    const params: Record<string, string | number> = {};
    if (kind === "new_monthly_obligation") {
      params.monthly_amount = normalizeAmount(monthlyAmount);
      params.start_date = startDate;
    } else if (kind === "income_gap") {
      params.monthly_income_delta = normalizeAmount(incomeDelta);
      params.months = Number.parseInt(months, 10) || 1;
    } else {
      params.one_off_amount = normalizeAmount(oneOff);
      params.goal_amount = normalizeAmount(goal);
      params.by_date = byDate;
    }
    try {
      await run.mutateAsync({
        path: { id: forecastId },
        body: { kind, params },
      });
    } catch (err) {
      setError(forecastActionErrorMessage(err));
    }
  };

  const kindLabel = useMemo(() => {
    switch (kind) {
      case "new_monthly_obligation":
        return "new monthly cost";
      case "income_gap":
        return "income change";
      default:
        return "one-off + goal";
    }
  }, [kind]);

  return (
    <section className="flex flex-col gap-5">
      <p className="text-sm text-muted-foreground">
        Check whether you can afford an extra cost (or income change) on top of
        your forecast.
      </p>

      <label className="flex max-w-sm flex-col gap-1.5 text-xs text-muted-foreground">
        What if I add…
        <select
          className="h-10 rounded-lg border border-input bg-background px-3 text-sm text-foreground"
          value={kind}
          disabled={run.isPending || assumptionsDirty}
          onChange={(e) =>
            setKind(
              e.target.value as
                | "new_monthly_obligation"
                | "income_gap"
                | "one_off_plus_goal",
            )
          }
        >
          <option value="new_monthly_obligation">
            an additional monthly cost
          </option>
          <option value="income_gap">a change in monthly income</option>
          <option value="one_off_plus_goal">
            a one-off cost and a savings goal
          </option>
        </select>
      </label>

      {kind === "new_monthly_obligation" ? (
        <div className="flex flex-col gap-3 sm:flex-row">
          <Field
            label="Monthly amount"
            value={monthlyAmount}
            onChange={setMonthlyAmount}
            disabled={run.isPending || assumptionsDirty}
          />
          <Field
            label="Start date"
            value={startDate}
            onChange={setStartDate}
            disabled={run.isPending || assumptionsDirty}
            type="date"
          />
        </div>
      ) : null}
      {kind === "income_gap" ? (
        <div className="flex flex-col gap-3 sm:flex-row">
          <Field
            label="Monthly income delta"
            value={incomeDelta}
            onChange={setIncomeDelta}
            disabled={run.isPending || assumptionsDirty}
          />
          <Field
            label="Months"
            value={months}
            onChange={setMonths}
            disabled={run.isPending || assumptionsDirty}
          />
        </div>
      ) : null}
      {kind === "one_off_plus_goal" ? (
        <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap">
          <Field
            label="One-off cost"
            value={oneOff}
            onChange={setOneOff}
            disabled={run.isPending || assumptionsDirty}
          />
          <Field
            label="Goal reserve"
            value={goal}
            onChange={setGoal}
            disabled={run.isPending || assumptionsDirty}
          />
          <Field
            label="By date"
            value={byDate}
            onChange={setByDate}
            disabled={run.isPending || assumptionsDirty}
            type="date"
          />
        </div>
      ) : null}

      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}

      <div className="flex justify-end pt-2">
        <Button
          type="button"
          disabled={run.isPending || assumptionsDirty}
          onClick={onRun}
        >
          {run.isPending ? "Comparing…" : `Compare ${kindLabel}`}
        </Button>
      </div>

      {scenarios.length > 0 ? (
        <ul className="mt-2 divide-y border-y">
          {scenarios.map((s) => (
            <ScenarioRow key={s.id} scenario={s} />
          ))}
        </ul>
      ) : null}
    </section>
  );
};

const ScenarioRow: React.FC<{ scenario: Scenario }> = ({ scenario }) => {
  const r = scenario.result;
  const create = useCreateDecision();
  const [error, setError] = useState<string | null>(null);
  const [chosen, setChosen] = useState(false);

  const onChoose = async () => {
    setError(null);
    const annual = annualFromMonthlyDelta(r.free_cashflow_delta);
    try {
      await create.mutateAsync({
        body: {
          scenario_id: scenario.id,
          title: `Act on ${scenarioKindLabel(scenario.kind)}`,
          assumptions: {
            scenario_kind: scenario.kind,
            free_cashflow_delta: r.free_cashflow_delta,
          },
          action: {
            title: actionTitleForScenario(scenario.kind),
            expected_annual_effect: annual,
            due_on: dueInDays(30),
          },
        },
      });
      setChosen(true);
    } catch (err) {
      setError(decisionActionErrorMessage(err));
    }
  };

  return (
    <li className="space-y-2 py-4">
      <p className="text-sm font-medium">{scenarioKindLabel(scenario.kind)}</p>
      <p className="text-xs text-muted-foreground">
        Min {formatAmount(r.min_balance)} · Ending {formatAmount(r.ending_balance)}
        {r.free_cashflow_delta !== "0.00"
          ? ` · Free cashflow Δ ${formatAmount(r.free_cashflow_delta)}`
          : null}
        {r.goal_feasible != null
          ? r.goal_feasible
            ? " · Goal feasible"
            : " · Goal not feasible"
          : null}
      </p>
      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}
      <div className="flex justify-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={create.isPending || chosen}
          onClick={onChoose}
        >
          {chosen
            ? "Action chosen"
            : create.isPending
              ? "Saving…"
              : "Choose this action"}
        </Button>
      </div>
    </li>
  );
};

const OpenActionsPanel: React.FC = () => {
  const query = useOpenActions();
  const update = useUpdateActionStatus();
  const [error, setError] = useState<string | null>(null);
  const actions = query.data?.data ?? [];

  const onStatus = async (action: Action, status: "done" | "skipped") => {
    setError(null);
    try {
      await update.mutateAsync({
        path: { id: action.id },
        body: { status },
      });
    } catch (err) {
      setError(decisionActionErrorMessage(err));
    }
  };

  return (
    <section className="flex flex-col gap-4">
      <p className="text-sm text-muted-foreground">
        Open actions from reviews and what-ifs. Mark done when you followed
        through.
      </p>
      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}
      {query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : actions.length === 0 ? (
        <p className="text-sm text-muted-foreground">No open actions yet.</p>
      ) : (
        <ul className="divide-y border-y">
          {actions.map((action) => (
            <li
              key={action.id}
              className="flex flex-col gap-2 py-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="min-w-0 space-y-1">
                <p className="text-sm font-medium">{action.title}</p>
                <p className="text-xs text-muted-foreground">
                  Due {formatDate(action.due_on)} · Expected{" "}
                  {formatAmount(action.expected_annual_effect)} / year
                </p>
              </div>
              <div className="flex shrink-0 gap-2">
                <Button
                  type="button"
                  size="sm"
                  disabled={update.isPending}
                  onClick={() => onStatus(action, "done")}
                >
                  Done
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={update.isPending}
                  onClick={() => onStatus(action, "skipped")}
                >
                  Skip
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};

function actionTitleForScenario(kind: string): string {
  switch (kind) {
    case "new_monthly_obligation":
      return "Decide on the additional monthly cost";
    case "income_gap":
      return "Plan for the income change";
    case "one_off_plus_goal":
      return "Fund the one-off cost and savings goal";
    default:
      return "Follow through on this what-if";
  }
}

function annualFromMonthlyDelta(delta: string): string {
  const monthly = Number.parseFloat(delta);
  if (Number.isNaN(monthly)) {
    return "0.00";
  }
  return (monthly * 12).toFixed(2);
}

const Stat: React.FC<{
  label: string;
  hint?: string;
  value: string;
  className?: string;
}> = ({ label, hint, value, className }) => (
  <div>
    <p className="text-xs font-medium text-foreground">{label}</p>
    {hint ? (
      <p className="text-xs text-muted-foreground">{hint}</p>
    ) : null}
    <p className={cn("mt-1 text-lg font-semibold tabular-nums", className)}>
      {value}
    </p>
  </div>
);

const Field: React.FC<{
  label: string;
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
  type?: string;
}> = ({ label, value, onChange, disabled, type = "text" }) => (
  <label className="flex min-w-[10rem] flex-1 flex-col gap-1.5 text-xs text-muted-foreground">
    {label}
    <input
      type={type}
      className="h-10 rounded-lg border border-input bg-background px-3 text-sm tabular-nums text-foreground"
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value)}
    />
  </label>
);

function scenarioKindLabel(kind: string): string {
  switch (kind) {
    case "new_monthly_obligation":
      return "New monthly cost";
    case "income_gap":
      return "Income change";
    case "one_off_plus_goal":
      return "One-off + savings goal";
    default:
      return kind;
  }
}

function formatAmount(value: string): string {
  const amount = Number.parseFloat(value);
  if (Number.isNaN(amount)) {
    return `${value} €`;
  }
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(amount);
}

function formatDate(value: string): string {
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.${y}`;
}

function normalizeAmount(value: string): string {
  const cleaned = value.trim().replace(",", ".");
  const amount = Number.parseFloat(cleaned);
  if (Number.isNaN(amount)) {
    return cleaned;
  }
  return amount.toFixed(2);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

export default PlanPage;
