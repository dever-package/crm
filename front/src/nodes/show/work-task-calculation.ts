import { useEffect, useMemo, useRef, useState } from "react";

import {
  applyWorkTaskRawValues,
  collectWorkTaskSubmitValues,
  errorMessage,
  positiveTextID,
  setWorkStoreValue,
  workApi,
  workTaskCalculationPath,
  workTaskFieldMapPath,
  workTaskFormDataPath,
  type WorkFormCalculationResponse,
  type WorkFormCalculationState,
  type WorkStoreLike,
  type WorkTask,
} from "./work-core";
import {
  emptyWorkTaskRecord,
  useWorkTaskStoreValue,
} from "./work-task-form-fields";

const idleCalculationState: WorkFormCalculationState = { status: "idle" };

export function useWorkTaskFormCalculation({
  store,
  task,
  customerID,
  assetID,
}: {
  store?: WorkStoreLike;
  task: WorkTask | null;
  customerID: string;
  assetID: string;
}) {
  const formValues = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const fieldMap = useWorkTaskStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    emptyWorkTaskRecord,
  );
  const scriptID = positiveTextID(task?.form?.calculation_script_id);
  const taskID = positiveTextID(task?.id);
  const todoID = positiveTextID(task?.todo_id);
  const workflowInstanceID = positiveTextID(task?.workflow_instance_id);
  const taskIdentity = [
    taskID,
    todoID,
    scriptID,
    customerID,
    assetID,
  ].join(":");
  const [outputKeys, setOutputKeys] = useState<string[]>([]);
  const lastCalculatedInputRef = useRef("");
  const requestSequenceRef = useRef(0);
  const sourceValues = useMemo(() => {
    const values = collectWorkTaskSubmitValues(store);
    for (const outputKey of outputKeys) {
      delete values[outputKey];
    }
    return values;
  }, [fieldMap, formValues, outputKeys, store]);
  const sourceSignature = useMemo(
    () => JSON.stringify(sourceValues),
    [sourceValues],
  );

  useEffect(() => {
    setOutputKeys([]);
    lastCalculatedInputRef.current = "";
    requestSequenceRef.current += 1;
    setWorkStoreValue(store, workTaskCalculationPath, idleCalculationState);
  }, [store, taskIdentity]);

  useEffect(() => {
    if (!store || !taskID || !todoID || !scriptID) return;
    if (sourceSignature === lastCalculatedInputRef.current) return;
    const requestValues = JSON.parse(sourceSignature) as Record<
      string,
      unknown
    >;
    const requestSequence = requestSequenceRef.current + 1;
    requestSequenceRef.current = requestSequence;
    const timer = window.setTimeout(async () => {
      const previous = currentWorkTaskCalculation(store);
      setWorkStoreValue(store, workTaskCalculationPath, {
        ...previous,
        status: "calculating",
        error: "",
      } satisfies WorkFormCalculationState);
      try {
        const response = await workApi<WorkFormCalculationResponse>(
          "/crm/work/calculate_form",
          {
            method: "POST",
            body: JSON.stringify({
              task_id: taskID,
              todo_id: todoID,
              workflow_instance_id: workflowInstanceID || undefined,
              customer_id: customerID || undefined,
              asset_id: assetID || undefined,
              values: requestValues,
            }),
          },
        );
        if (requestSequenceRef.current !== requestSequence) return;
        const outputFields = response.fields || {};
        const nextOutputKeys = Object.keys(outputFields);
        setOutputKeys(nextOutputKeys);
        const calculationInput = { ...requestValues };
        for (const outputKey of nextOutputKeys) {
          delete calculationInput[outputKey];
        }
        lastCalculatedInputRef.current = JSON.stringify(calculationInput);
        applyWorkTaskRawValues(store, outputFields);
        setWorkStoreValue(store, workTaskCalculationPath, {
          ...response,
          status: response.passed ? "success" : "incomplete",
          error: "",
        } satisfies WorkFormCalculationState);
      } catch (error) {
        if (requestSequenceRef.current !== requestSequence) return;
        setWorkStoreValue(store, workTaskCalculationPath, {
          status: "error",
          error: errorMessage(error, "自动计算失败"),
        } satisfies WorkFormCalculationState);
      }
    }, 280);
    return () => window.clearTimeout(timer);
  }, [
    assetID,
    customerID,
    scriptID,
    sourceSignature,
    store,
    taskID,
    todoID,
    workflowInstanceID,
  ]);
}

function currentWorkTaskCalculation(
  store: WorkStoreLike | undefined,
): WorkFormCalculationState {
  const value = (store as {
    getState?: () => { data?: { actionTarget?: Record<string, unknown> } };
  })?.getState?.()?.data?.actionTarget?.workTaskCalculation;
  return value && typeof value === "object"
    ? (value as WorkFormCalculationState)
    : idleCalculationState;
}
