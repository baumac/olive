import type * as React from 'react';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'md' | 'sm';
}
export declare function Button(props: ButtonProps): React.ReactElement;

export interface TextFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: React.ReactNode;
  hint?: React.ReactNode;
  error?: React.ReactNode;
  multiline?: boolean;
}
export declare function TextField(props: TextFieldProps): React.ReactElement;

export interface SelectOption { value: string; label: string; disabled?: boolean }
export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label: React.ReactNode;
  options: SelectOption[];
  placeholder?: string;
  hint?: React.ReactNode;
  error?: React.ReactNode;
}
export declare function Select(props: SelectProps): React.ReactElement;

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: React.ReactNode;
  hint?: React.ReactNode;
}
export declare function Checkbox(props: CheckboxProps): React.ReactElement;

export interface CardProps extends Omit<React.HTMLAttributes<HTMLElement>, 'title'> {
  title?: React.ReactNode;
  actions?: React.ReactNode;
  footer?: React.ReactNode;
}
export declare function Card(props: CardProps): React.ReactElement;

export interface TabItem { id: string; label: React.ReactNode; content: React.ReactNode }
export interface TabsProps {
  tabs: TabItem[];
  label: string;
  value?: string;
  defaultValue?: string;
  onChange?: (id: string) => void;
  className?: string;
}
export declare function Tabs(props: TabsProps): React.ReactElement;

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: 'neutral' | 'brand' | 'success' | 'danger';
}
export declare function Badge(props: BadgeProps): React.ReactElement;

export interface AlertProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'title'> {
  tone?: 'info' | 'success' | 'danger';
  title: React.ReactNode;
}
export declare function Alert(props: AlertProps): React.ReactElement;

declare global {
  interface Window {
    Sponga47: {
      Button: typeof Button; TextField: typeof TextField; Select: typeof Select; Checkbox: typeof Checkbox;
      Card: typeof Card; Tabs: typeof Tabs; Badge: typeof Badge; Alert: typeof Alert;
    };
  }
}
