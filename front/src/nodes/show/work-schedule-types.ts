export type WorkScheduleType = "customer_follow" | "personal";
export type WorkScheduleStatus = "pending" | "completed" | "canceled";
export type WorkScheduleView = "timeGridDay" | "timeGridWeek" | "dayGridMonth";

export type WorkScheduleEvent = {
  id?: string | number;
  schedule_type?: WorkScheduleType;
  customer_id?: string | number;
  customer_name?: string;
  customer_phone?: string;
  owner_staff_id?: string | number;
  source_workflow_instance_id?: string | number;
  title?: string;
  remark?: string;
  start_at?: string;
  end_at?: string;
  reminder_minutes?: string | number;
  remind_at?: string;
  source?: string;
  status?: WorkScheduleStatus;
  can_edit?: boolean;
  participant_ids?: Array<string | number>;
  resource_ids?: Array<string | number>;
};

export type WorkScheduleCustomerOption = {
  id?: string | number;
  name?: string;
  phone?: string;
  owner_staff_id?: string | number;
  owner_staff_name?: string;
  next_follow_at?: string;
  schedule_event_id?: string | number;
};

export type WorkScheduleStaffOption = {
  id?: string | number;
  name?: string;
  department_id?: string | number;
};

export type WorkScheduleResourceOption = {
  id?: string | number;
  name?: string;
  location?: string;
  capacity?: string | number;
};

export type WorkScheduleReminderOption = {
  id?: string | number;
  value?: string;
};

export type WorkScheduleOptions = {
  customers?: WorkScheduleCustomerOption[];
  staff?: WorkScheduleStaffOption[];
  resources?: WorkScheduleResourceOption[];
  reminders?: WorkScheduleReminderOption[];
  config?: {
    ready?: boolean;
    message?: string;
    template_name?: string;
    field_name?: string;
  };
};

export type WorkScheduleListResponse = {
  list?: WorkScheduleEvent[];
  total?: number;
  range_start?: string;
  range_end?: string;
};

export type WorkScheduleReminderResponse = {
  list?: WorkScheduleEvent[];
  total?: number;
};

export type WorkScheduleRange = {
  start: Date;
  end: Date;
};

export type WorkScheduleDraft = {
  scheduleType: WorkScheduleType;
  customerID: string;
  title: string;
  remark: string;
  startAt: string;
  endAt: string;
  reminderMinutes: string;
  participantIDs: string[];
  resourceIDs: string[];
};
