export type WorkStaffOption = {
  id?: string | number;
  name?: string;
  phone?: string;
  department_id?: string | number;
};

export type WorkDepartmentOption = {
  id?: string | number;
  name?: string;
};

export type WorkPersonSnapshot = {
  staff_id?: string | number;
  staff_name?: string;
  phone?: string;
  department_id?: string | number;
  department_name?: string;
};

export type WorkPeopleOptions = {
  staff?: WorkStaffOption[];
  departments?: WorkDepartmentOption[];
  current_staff_id?: string | number;
  current_department_id?: string | number;
};
