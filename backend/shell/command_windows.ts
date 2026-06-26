// Hide the child console window on Windows via windowsHide at spawn time.
export const set_hide_window = (cmd: { windows_hide?: boolean }): void => {
  cmd.windows_hide = true;
};

