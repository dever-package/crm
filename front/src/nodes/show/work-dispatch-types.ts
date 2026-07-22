export type DispatchDepartment = {
  id?: string | number;
  name?: string;
  code?: string;
};

export type DispatchRoute = {
  workflow_id?: string | number;
  workflow_name?: string;
  source_stage_id?: string | number;
  source_stage_name?: string;
  source_department_id?: string | number;
  source_department_name?: string;
  target_workflow_id?: string | number;
  target_workflow_name?: string;
  target_stage_id?: string | number;
  target_stage_name?: string;
  target_department_id?: string | number;
  target_department_name?: string;
};

export type DispatchStaff = {
  id?: string | number;
  name?: string;
  phone?: string;
  staff_type?: string;
  status?: string | number;
  today_auto_count?: string | number;
};

export type DispatchSchedule = Record<string, Array<[number, number]>>;

export type DispatchMember = {
  id?: string | number;
  staff_id?: string | number;
  daily_limit?: string | number;
  status?: string | number;
  sort?: string | number;
  weekly_schedule?: DispatchSchedule;
  weekly_schedule_json?: string;
  today_auto_count?: string | number;
};

export type DispatchPool = {
  id?: string | number;
  name?: string;
  pool_type?: "direct" | "group" | string;
  is_active?: boolean;
  member_list?: DispatchMember[];
};

export type PendingDispatchRow = {
  kind?: "stage" | "task" | string;
  id?: string | number;
  todo_id?: string | number;
  workflow_instance_id?: string | number;
  title?: string;
  subject_name?: string;
  subject_no?: string;
  asset_name?: string;
  asset_no?: string;
  created_at?: string;
  handoff_id?: string | number;
  lead_id?: string | number;
  lead_name?: string;
  lead_code?: string;
  phone?: string;
  source_stage_id?: string | number;
  target_stage_id?: string | number;
  target_department_id?: string | number;
};

export type DispatchActiveLead = {
  workflow_instance_id?: string | number;
  lead_id?: string | number;
  lead_name?: string;
  lead_code?: string;
  phone?: string;
  stage_id?: string | number;
  stage_name?: string;
  owner_staff_id?: string | number;
  owner_staff_name?: string;
  started_at?: string;
  updated_at?: string;
};

export type DispatchActiveLeadPayload = {
  department_id?: string | number;
  list?: DispatchActiveLead[];
  owner_options?: DispatchStaff[];
  total?: string | number;
  page?: string | number;
  page_size?: string | number;
};

export type DispatchBatchReassignResult = {
  success?: boolean;
  selected_count?: string | number;
  changed_count?: string | number;
};

export type DispatchConfigPayload = {
  can_manage?: boolean;
  is_global?: boolean;
  department_id?: string | number;
  department_name?: string;
  departments?: DispatchDepartment[];
  staff?: DispatchStaff[];
  active_pool_id?: string | number;
  version?: string | number;
  pools?: DispatchPool[];
  pending?: PendingDispatchRow[];
  pending_count?: string | number;
  retry_warning?: string;
  workflow_id?: string | number;
  workflow_name?: string;
  source_stage_id?: string | number;
  source_stage_name?: string;
  source_department_id?: string | number;
  source_department_name?: string;
  target_workflow_id?: string | number;
  target_workflow_name?: string;
  target_stage_id?: string | number;
  target_stage_name?: string;
  target_department_id?: string | number;
  target_department_name?: string;
  routes?: DispatchRoute[];
  auto_handoff_enabled?: boolean;
  assignee_options?: DispatchStaff[];
};

export type DispatchAssignResult = {
  success?: boolean;
  selected_count?: string | number;
  assigned_count?: string | number;
};

export type DispatchMemberDraft = {
  staffId: string;
  enabled: boolean;
  dailyLimit: number;
  schedule: DispatchSchedule;
};

export const fullWeekDispatchSchedule = (): DispatchSchedule =>
  Object.fromEntries(
    Array.from({ length: 7 }, (_, index) => [
      String(index + 1),
      [[0, 1440] as [number, number]],
    ]),
  );

export function cloneDispatchSchedule(
  schedule?: DispatchSchedule,
): DispatchSchedule {
  const source = schedule && Object.keys(schedule).length
    ? schedule
    : fullWeekDispatchSchedule();
  return Object.fromEntries(
    Array.from({ length: 7 }, (_, index) => {
      const day = String(index + 1);
      return [day, (source[day] || []).map((period) => [...period] as [number, number])];
    }),
  );
}
