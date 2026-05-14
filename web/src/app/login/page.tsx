import { LoginForm } from "@/features/auth/login-form";

export default function LoginPage() {
  return (
    <div className="relative flex min-h-screen flex-col bg-gradient-to-b from-background via-muted/25 to-background">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-40 bg-[radial-gradient(circle_at_top,_rgba(148,163,184,0.35),_transparent_55%)] dark:bg-[radial-gradient(circle_at_top,_rgba(71,85,105,0.45),_transparent_55%)]" />
      <div className="relative flex flex-1 items-center justify-center px-4 py-14">
        <LoginForm />
      </div>
    </div>
  );
}
