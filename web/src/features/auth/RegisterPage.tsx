import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { UserPlus } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { authApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { useAuthStore } from "./authStore";
import { AuthLayout } from "./AuthLayout";

export function RegisterPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.setSession);
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setServerError(null);
    const next: Record<string, string> = {};
    if (!displayName.trim()) next.displayName = t("auth.validation.displayName");
    if (!/^\S+@\S+\.\S+$/.test(email)) next.email = t("auth.validation.email");
    if (password.length < 8) next.password = t("auth.validation.password");
    setErrors(next);
    if (Object.keys(next).length > 0) return;

    setSubmitting(true);
    try {
      const res = await authApi.register({
        email,
        password,
        display_name: displayName.trim(),
      });
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
        <h2>{t("auth.register.title")}</h2>
        <div className="field">
          <label className="field-label" htmlFor="displayName">
            {t("auth.displayName")}
          </label>
          <input
            id="displayName"
            className="input"
            value={displayName}
            maxLength={60}
            onChange={(e) => setDisplayName(e.target.value)}
            required
          />
          {errors.displayName && (
            <span className="field-error">{errors.displayName}</span>
          )}
        </div>
        <div className="field">
          <label className="field-label" htmlFor="email">
            {t("auth.email")}
          </label>
          <input
            id="email"
            className="input"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
          {errors.email && <span className="field-error">{errors.email}</span>}
        </div>
        <div className="field">
          <label className="field-label" htmlFor="password">
            {t("auth.password")}
          </label>
          <input
            id="password"
            className="input"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {errors.password && (
            <span className="field-error">{errors.password}</span>
          )}
        </div>
        {serverError && (
          <p className="field-error" role="alert">
            {serverError}
          </p>
        )}
        <button className="btn btn-primary" type="submit" disabled={submitting}>
          <UserPlus size={15} aria-hidden />
          {t("auth.register.cta")}
        </button>
        <Link to="/login" className="faint">
          {t("auth.toLogin")}
        </Link>
      </form>
    </AuthLayout>
  );
}
