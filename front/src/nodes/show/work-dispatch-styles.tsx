export function WorkDispatchStyles() {
  return (
    <style>{`
      .crm-dispatch-page { min-width: 0; color: #171a19; }
      .crm-dispatch-header { display: flex; min-height: 64px; align-items: center; justify-content: space-between; gap: 16px; border-bottom: 1px solid #e4e8e6; padding: 14px 20px; }
      .crm-dispatch-header > div:first-child { min-width: 0; }
      .crm-dispatch-header h2 { margin: 0; font-size: 17px; font-weight: 700; line-height: 1.35; }
      .crm-dispatch-header p { margin: 3px 0 0; color: #6b7370; font-size: 12px; }
      .crm-dispatch-header-actions { display: flex; align-items: center; gap: 8px; }
      .crm-dispatch-department { min-width: 168px; height: 34px; border: 1px solid #d8dedb; border-radius: 6px; background: #fff; padding: 0 10px; font: inherit; }
      .crm-dispatch-route { display: grid; grid-template-columns: minmax(170px, 1fr) 28px minmax(170px, 1fr) minmax(220px, 1.4fr); align-items: center; gap: 16px; border-bottom: 1px solid #e4e8e6; background: #f8faf9; padding: 14px 20px; }
      .crm-dispatch-route > svg { color: #7a827f; }
      .crm-dispatch-route-point { display: grid; min-width: 0; grid-template-columns: auto minmax(0, 1fr); align-items: baseline; gap: 2px 8px; }
      .crm-dispatch-route-point small { grid-row: 1 / 3; color: #7a827f; font-size: 11px; }
      .crm-dispatch-route-point strong, .crm-dispatch-route-point span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-route-point strong { font-size: 13px; }
      .crm-dispatch-route-point span { color: #66706c; font-size: 12px; }
      .crm-dispatch-route-switch { display: flex; min-width: 0; align-items: center; justify-content: flex-end; gap: 12px; }
      .crm-dispatch-route-switch > span { display: flex; min-width: 0; flex-direction: column; text-align: right; }
      .crm-dispatch-route-switch strong { font-size: 13px; }
      .crm-dispatch-route-switch small { color: #747d79; font-size: 11px; }
      .crm-dispatch-tabs { display: flex; gap: 2px; border-bottom: 1px solid #e4e8e6; padding: 0 20px; }
      .crm-dispatch-tabs button { position: relative; min-height: 44px; border-radius: 0; padding: 0 16px; }
      .crm-dispatch-tabs button.is-active { color: #0f766e; }
      .crm-dispatch-tabs button.is-active::after { position: absolute; right: 12px; bottom: -1px; left: 12px; height: 2px; background: #0f766e; content: ""; }
      .crm-dispatch-workspace { display: grid; grid-template-columns: minmax(0, 1fr) 248px; min-height: 420px; }
      .crm-dispatch-main { min-width: 0; padding: 18px 20px 22px; }
      .crm-dispatch-side { border-left: 1px solid #e4e8e6; padding: 18px 16px; background: #fafbfa; }
      .crm-dispatch-section-head { display: flex; min-height: 36px; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
      .crm-dispatch-section-head strong { font-size: 14px; }
      .crm-dispatch-section-head span { color: #6b7370; font-size: 12px; }
      .crm-dispatch-current { display: inline-flex; align-items: center; gap: 5px; color: #0f766e; }
      .crm-dispatch-current::before { width: 6px; height: 6px; border-radius: 50%; background: #0f766e; content: ""; }
      .crm-dispatch-add { position: relative; margin-bottom: 14px; }
      .crm-dispatch-add-results { position: absolute; z-index: 20; top: calc(100% + 4px); right: 0; left: 0; max-height: 230px; overflow-y: auto; border: 1px solid #d8dedb; border-radius: 6px; background: #fff; box-shadow: 0 12px 28px rgb(23 26 25 / 12%); padding: 4px; }
      .crm-dispatch-add-results button { display: flex; width: 100%; min-height: 38px; align-items: center; justify-content: space-between; border: 0; border-radius: 4px; background: transparent; padding: 7px 9px; text-align: left; }
      .crm-dispatch-add-results button:hover { background: #f1f4f2; }
      .crm-dispatch-add-results small { color: #7a827f; }
      .crm-dispatch-members { border-top: 1px solid #e4e8e6; }
      .crm-dispatch-member { display: grid; grid-template-columns: 56px minmax(130px, 1fr) 116px 152px 90px; min-height: 66px; align-items: center; gap: 12px; border-bottom: 1px solid #e4e8e6; }
      .crm-dispatch-order { display: flex; align-items: center; gap: 2px; color: #6b7370; }
      .crm-dispatch-order button { width: 24px; height: 24px; padding: 0; }
      .crm-dispatch-member-person { min-width: 0; }
      .crm-dispatch-member-person strong, .crm-dispatch-member-person span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-member-person span { margin-top: 3px; color: #747d79; font-size: 11px; }
      .crm-dispatch-limit { display: flex; align-items: center; gap: 6px; }
      .crm-dispatch-limit input { width: 70px; }
      .crm-dispatch-limit small { color: #747d79; white-space: nowrap; }
      .crm-dispatch-member-actions { display: flex; justify-content: flex-end; gap: 3px; }
      .crm-dispatch-toggle { position: relative; width: 34px; height: 19px; border: 0; border-radius: 10px; background: #cfd6d2; padding: 0; transition: background 120ms ease; }
      .crm-dispatch-toggle::after { position: absolute; top: 2px; left: 2px; width: 15px; height: 15px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgb(0 0 0 / 18%); content: ""; transition: transform 120ms ease; }
      .crm-dispatch-toggle.is-on { background: #0f766e; }
      .crm-dispatch-toggle.is-on::after { transform: translateX(15px); }
      .crm-dispatch-groups { display: flex; flex-direction: column; gap: 4px; }
      .crm-dispatch-group { display: flex; width: 100%; min-height: 40px; align-items: center; justify-content: space-between; border: 1px solid transparent; border-radius: 6px; background: transparent; padding: 0 10px; text-align: left; }
      .crm-dispatch-group:hover { background: #f0f3f1; }
      .crm-dispatch-group.is-selected { border-color: #cbd7d2; background: #e9f1ee; }
      .crm-dispatch-group span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-group small { color: #68716d; }
      .crm-dispatch-empty { display: grid; min-height: 160px; place-items: center; color: #747d79; text-align: center; }
      .crm-dispatch-pending { border-top: 8px solid #f4f6f5; padding: 18px 20px 30px; }
      .crm-dispatch-pending-toolbar { display: flex; align-items: center; justify-content: flex-end; gap: 8px; margin-bottom: 12px; }
      .crm-dispatch-pending-toolbar > span { margin-right: auto; color: #68716d; font-size: 12px; }
      .crm-dispatch-pending-toolbar select, .crm-dispatch-pending-row select { height: 32px; border: 1px solid #d8dedb; border-radius: 6px; background: #fff; padding: 0 8px; font: inherit; }
      .crm-dispatch-pending-toolbar select { width: 174px; }
      .crm-dispatch-pending-list { min-width: 0; overflow-x: auto; border-top: 1px solid #e4e8e6; }
      .crm-dispatch-pending-row { display: grid; min-width: 920px; grid-template-columns: 32px minmax(210px, 1.2fr) minmax(180px, 1fr) 150px 174px 76px; min-height: 58px; align-items: center; gap: 12px; border-bottom: 1px solid #e4e8e6; }
      .crm-dispatch-pending-row.is-head { min-height: 42px; background: #f8faf9; color: #59625e; font-size: 12px; font-weight: 600; }
      .crm-dispatch-pending-row input[type="checkbox"] { width: 16px; height: 16px; margin: 0; accent-color: #0f766e; }
      .crm-dispatch-pending-row > span { overflow: hidden; color: #4f5854; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-pending-lead { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
      .crm-dispatch-pending-lead strong, .crm-dispatch-pending-lead small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-pending-lead small { color: #747d79; }
      .crm-dispatch-pending-row select { width: 100%; }
      .crm-dispatch-active { border-top: 8px solid #f4f6f5; padding: 18px 20px 22px; }
      .crm-dispatch-active-toolbar { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 12px; margin-bottom: 14px; }
      .crm-dispatch-active-filters, .crm-dispatch-active-batch { display: flex; min-width: 0; align-items: center; gap: 8px; }
      .crm-dispatch-active-filters { max-width: 720px; }
      .crm-dispatch-active-filters select, .crm-dispatch-active-batch select { height: 36px; border: 1px solid #d8dedb; border-radius: 6px; background: #fff; padding: 0 10px; font: inherit; }
      .crm-dispatch-active-filters select { width: 154px; }
      .crm-dispatch-active-batch { justify-content: flex-end; }
      .crm-dispatch-active-batch > span { color: #68716d; font-size: 12px; white-space: nowrap; }
      .crm-dispatch-active-batch select { width: 164px; }
      .crm-dispatch-active-list { min-width: 0; overflow-x: auto; border-top: 1px solid #e4e8e6; }
      .crm-dispatch-active-row { display: grid; min-width: 760px; grid-template-columns: 32px minmax(220px, 1.4fr) minmax(120px, .7fr) minmax(130px, .7fr) minmax(150px, .8fr); min-height: 58px; align-items: center; gap: 12px; border-bottom: 1px solid #e4e8e6; }
      .crm-dispatch-active-row.is-head { min-height: 42px; background: #f8faf9; color: #59625e; font-size: 12px; font-weight: 600; }
      .crm-dispatch-active-row input[type="checkbox"] { width: 16px; height: 16px; margin: 0; accent-color: #0f766e; }
      .crm-dispatch-active-row > span { min-width: 0; overflow: hidden; color: #4f5854; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-active-lead { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
      .crm-dispatch-active-lead strong, .crm-dispatch-active-lead small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .crm-dispatch-active-lead small { color: #747d79; }
      .crm-dispatch-schedule-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
      .crm-dispatch-schedule-toolbar > div { display: flex; gap: 6px; }
      .crm-dispatch-schedule-toolbar span { display: inline-flex; align-items: center; gap: 5px; color: #68716d; font-size: 12px; }
      .crm-dispatch-schedule-scroll { overflow-x: auto; padding: 8px 0 4px; }
      .crm-dispatch-schedule-grid { display: grid; width: 100%; min-width: 744px; grid-template-columns: 54px repeat(24, minmax(25px, 1fr)); border-top: 1px solid #dce2df; border-left: 1px solid #dce2df; }
      .crm-dispatch-schedule-grid > * { min-width: 0; height: 28px; border-right: 1px solid #dce2df; border-bottom: 1px solid #dce2df; }
      .crm-dispatch-schedule-hour { display: grid; place-items: center; background: #f7f9f8; color: #78817d; font-size: 10px; }
      .crm-dispatch-schedule-day { display: grid; place-items: center; background: #f7f9f8; font-size: 11px; font-weight: 500; }
      .crm-dispatch-schedule-grid button { border-top: 0; border-bottom: 1px solid #dce2df; border-left: 0; border-right: 1px solid #dce2df; background: #fff; padding: 0; }
      .crm-dispatch-schedule-grid button:hover { background: #d7ebe6; }
      .crm-dispatch-schedule-grid button.is-selected { background: #16867c; }
      @media (max-width: 1050px) {
        .crm-dispatch-route { grid-template-columns: minmax(150px, 1fr) 24px minmax(150px, 1fr); }
        .crm-dispatch-route-switch { grid-column: 1 / -1; justify-content: flex-start; border-top: 1px solid #e4e8e6; padding-top: 10px; }
        .crm-dispatch-route-switch > span { text-align: left; }
        .crm-dispatch-workspace { grid-template-columns: 1fr; }
        .crm-dispatch-side { order: -1; border-bottom: 1px solid #e4e8e6; border-left: 0; }
        .crm-dispatch-groups { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); }
        .crm-dispatch-member { grid-template-columns: 48px minmax(120px, 1fr) 105px 132px 84px; }
        .crm-dispatch-active-toolbar { grid-template-columns: 1fr; }
        .crm-dispatch-active-batch { justify-content: flex-start; }
      }
      @media (max-width: 760px) {
        .crm-dispatch-header { align-items: flex-start; flex-direction: column; }
        .crm-dispatch-header-actions { width: 100%; flex-wrap: wrap; }
        .crm-dispatch-department { min-width: 0; flex: 1; }
        .crm-dispatch-member { grid-template-columns: 42px minmax(0, 1fr) 92px; gap: 8px; padding: 10px 0; }
        .crm-dispatch-limit, .crm-dispatch-member-actions { grid-column: 2 / -1; justify-content: flex-start; }
        .crm-dispatch-route { grid-template-columns: minmax(0, 1fr) 20px minmax(0, 1fr); gap: 8px; }
        .crm-dispatch-pending-toolbar { align-items: stretch; flex-wrap: wrap; }
        .crm-dispatch-pending-toolbar > span { flex-basis: 100%; }
        .crm-dispatch-pending-toolbar select { min-width: 0; flex: 1; }
        .crm-dispatch-active-filters, .crm-dispatch-active-batch { align-items: stretch; flex-wrap: wrap; }
        .crm-dispatch-active-filters > div { flex-basis: 100%; }
        .crm-dispatch-active-filters select, .crm-dispatch-active-batch select { width: auto; min-width: 0; flex: 1; }
      }
    `}</style>
  );
}
