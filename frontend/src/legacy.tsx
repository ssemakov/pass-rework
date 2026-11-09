// Material icon paths used by legacy Pass (@material-ui/icons 4.9.1, MIT).
import { useId, type InputHTMLAttributes, type ReactNode } from "react";
const paths = {
  Add: "M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z",
  KeyboardReturn: "M19 7v4H5.83l3.58-3.59L8 6l-6 6 6 6 1.41-1.41L5.83 13H21V7z",
  FileCopy:
    "M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm-1 4l6 6v10c0 1.1-.9 2-2 2H7.99C6.89 23 6 22.1 6 21l.01-14c0-1.1.89-2 1.99-2h7zm-1 7h5.5L14 6.5V12z",
  Visibility:
    "M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z",
  VisibilityOff:
    "M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78l3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z",
  FolderOpen:
    "M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z",
  Edit: "M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34a.9959.9959 0 00-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z",
  Delete:
    "M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z",
  Close:
    "M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z",
} as const;
export function Icon({ name }: { name: keyof typeof paths }) {
  return (
    <svg
      className="material-icon"
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
    >
      <path d={paths[name]} />
    </svg>
  );
}
export function IconButton({
  name,
  label,
  tooltip = label,
  small = false,
  ...props
}: {
  name: keyof typeof paths;
  label: string;
  tooltip?: string;
  small?: boolean;
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <span className="tooltip-anchor">
      <button
        type="button"
        className={"icon-button" + (small ? " small" : "")}
        aria-label={label}
        {...props}
      >
        <Icon name={name} />
      </button>
      <span role="tooltip" className="tooltip">
        {tooltip}
      </span>
    </span>
  );
}
export function Field({
  label,
  dense = false,
  end,
  className = "",
  value,
  ...props
}: {
  label?: string;
  dense?: boolean;
  end?: ReactNode;
} & InputHTMLAttributes<HTMLInputElement>) {
  const generated = useId(),
    id = props.id ?? generated;
  return (
    <div
      className={
        "outlined-field " +
        (dense ? "dense " : "") +
        (label ? "labeled " : "") +
        (String(value ?? "").length ? "filled " : "") +
        className
      }
    >
      <input
        {...props}
        id={id}
        value={value}
        placeholder={label ? " " : props.placeholder}
      />
      {label && <label htmlFor={id}>{label}</label>}
      {end && <span className="input-adornment">{end}</span>}
      <fieldset aria-hidden="true">
        <legend>{label ? <span>{label}</span> : <span>&#8203;</span>}</legend>
      </fieldset>
    </div>
  );
}
