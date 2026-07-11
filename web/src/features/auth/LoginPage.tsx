import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { LogIn, AlertTriangle } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { authApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { Button } from "@/shared/ui/Button";
import { useAuthStore } from "./authStore";
import { AuthLayout } from "./AuthLayout";

export function LoginPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.setSession);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setServerError(null);
    if (!/^\S+@\S+\.\S+$/.test(email)) {
      setFieldError(t("auth.validation.email"));
      return;
    }
    setFieldError(null);
    setSubmitting(true);
    try {
      const res = await authApi.login({ email, password });
      setSession({ user: res.user, tokens: res.tokens });
      navigate("/app/dashboard");
    } catch (err) {
      setServerError(t(errorKey(err)));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthLayout>
      <form className="auth-form" onSubmit={onSubmit} noValidate>
        <h2>{t("auth.login.title")}</h2>
        <div className="field">
          <label className="field-label" htmlFor="email">
            {t("auth.email")}
          </label>
          <input
            id="email"
            className="input"
            type="email"
            autoComplete="email"
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
          {fieldError && <span className="field-error">{fieldError}</span>}
        </div>
        <div className="field">
          <label className="field-label" htmlFor="password">
            {t("auth.password")}
          </label>
          <input
            id="password"
            className="input"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        {serverError && (
          <p className="auth-error" role="alert">
            <AlertTriangle size={14} aria-hidden />
            {serverError}
          </p>
        )}
        <Button type="submit" loading={submitting} fullWidth>
          <LogIn size={15} aria-hidden />
          {t("auth.login.cta")}
        </Button>
        <Link to="/register" className="faint">
          {t("auth.toRegister")}
        </Link>
      </form>
    </AuthLayout>
  );
}
