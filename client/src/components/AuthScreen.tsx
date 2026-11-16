import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/index";
import { saveAuth } from "../store/auth";
import { generateKeyPair, exportPublicKey, storePrivateKey } from "../crypto/index";
import { ensureLocalDeviceRegistered } from "../features/e2ee";
import { apiErrorFields } from "../util/errors";
import { ArrowRight, Eye, EyeOff, Server, MessageCircle } from "lucide-react";
import LanguageSwitcher from "./LanguageSwitcher";
import ServerSettings from "./ServerSettings";

interface AuthScreenProps {
  onAuth: () => void;
  sessionNotice?: string | null;
}
type Tab = "login" | "register";

export default function AuthScreen({ onAuth, sessionNotice }: AuthScreenProps) {
  const { t } = useTranslation();
  const [tab, setTab]           = useState<Tab>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [totpCode, setTotpCode] = useState("");
  const [mfaRequired, setMfaRequired] = useState(false);
  const [error, setError]       = useState<string | null>(null);
  const [loading, setLoading]   = useState(false);
  const [showServers, setShowServers] = useState(false);
  const [usernameStatus, setUsernameStatus] = useState<"idle" | "checking" | "taken" | "available">("idle");
  const checkTimer = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => { if (checkTimer.current) clearTimeout(checkTimer.current); }, []);

  const selectTab = (next: Tab) => {
    if (checkTimer.current) clearTimeout(checkTimer.current);
    setTab(next); setError(null); setMfaRequired(false); setTotpCode('');
    setShowPassword(false); setUsernameStatus('idle');
  };

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Клиентская валидация
    const trimUser = username.trim();
    const trimPass = password.trim();

    if (trimUser.length < 2) {
      setError(t('auth.loginError'));
      return;
    }
    if (trimUser.length > 32) {
      setError(t('auth.loginError'));
      return;
    }
    if (!/^[a-zA-Z0-9_.-]+$/.test(trimUser)) {
      setError(t('auth.loginError'));
      return;
    }
    if (tab === "register" && usernameStatus === "taken") {
      setError(t('auth.registerError'));
      return;
    }
    if (tab === "register" && trimPass.length < 8) {
      setError(t('auth.registerError'));
      return;
    }

    setLoading(true);
    try {
      if (tab === "register") {
        const keyPair    = generateKeyPair();
        const pubKeyB64  = exportPublicKey(keyPair.publicKey);
        const res        = await api.auth.register(trimUser, password, pubKeyB64);
        await storePrivateKey(keyPair.secretKey);
        saveAuth(res.token, { id: res.user_id, username: trimUser, public_key: pubKeyB64 });
        await registerE2EEDeviceBestEffort(res.user_id);
      } else {
        const res = await api.auth.login(trimUser, password, totpCode || undefined);
        saveAuth(res.token, { id: res.user.id, username: res.user.username, public_key: res.user.public_key });
        await registerE2EEDeviceBestEffort(res.user.id);
      }
      onAuth();
    } catch (err: unknown) {
      const { code, message } = apiErrorFields(err);
      if (code === "mfa_required") { setMfaRequired(true); setTotpCode(""); setError(message ?? "Authenticator code required"); }
      else if (code === "username_taken") setError(t('auth.registerError'));
      else if (code === "invalid_credentials") setError(t('auth.loginError'));
      else if (code === "account_banned") setError(t('auth.loginError'));
      else setError(message ?? t('common.error'));
    } finally {
      setLoading(false);
    }
  }, [tab, username, password, totpCode, usernameStatus, onAuth, t]);

  return (
    <main className="auth-screen">
      <header className="auth-navigation">
        <a className="wordmark" href="#" aria-label="Zenthril"><span className="brand-symbol" aria-hidden="true">Z</span>Zenthril</a>
        <div className="auth-utilities">
          <button className="quiet-button" type="button" onClick={() => setShowServers(true)}><Server size={16} aria-hidden="true" /><span>{t('design.server')}</span></button>
          <LanguageSwitcher />
        </div>
      </header>
      <section className="auth-content" aria-labelledby="auth-title">
        <div className="auth-heading">
          <span className="auth-app-icon" aria-hidden="true"><MessageCircle size={32} strokeWidth={1.6} /></span>
          <p className="eyebrow">Zenthril Messenger</p>
          <h1 id="auth-title">{t(tab === 'login' ? 'design.welcome' : 'design.createAccount')}</h1>
          <p className="auth-description">{t(tab === 'login' ? 'design.signInDescription' : 'design.registerDescription')}</p>
        </div>
        <div className="auth-card">
          <div className="segmented-control" role="tablist" aria-label={t('design.accountAction')} onKeyDown={event => {
            if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
            event.preventDefault();
            const next: Tab = event.key === 'Home' ? 'login' : event.key === 'End' ? 'register' : tab === 'login' ? 'register' : 'login';
            selectTab(next);
            event.currentTarget.querySelector<HTMLButtonElement>(`#auth-${next}`)?.focus();
          }}>
            {(['login','register'] as Tab[]).map(item => <button key={item} type="button" role="tab" id={`auth-${item}`} tabIndex={tab === item ? 0 : -1} aria-selected={tab === item} aria-controls="auth-form" onClick={() => selectTab(item)}>
              {t(item === 'login' ? 'auth.login' : 'auth.register')}
            </button>)}
          </div>
          <form id="auth-form" role="tabpanel" aria-labelledby={`auth-${tab}`} onSubmit={handleSubmit} className="auth-form" aria-busy={loading}>
            <div className="form-field">
              <label htmlFor="auth-username">{t('auth.username')}</label>
              <input id="auth-username" name="username" value={username} autoComplete="username" autoCapitalize="none" spellCheck={false} required minLength={2} maxLength={32} aria-describedby={tab === 'register' ? 'username-hint' : undefined}
                onChange={event => {
                  const value = event.target.value; setUsername(value); setError(null); setUsernameStatus('idle');
                  if (checkTimer.current) clearTimeout(checkTimer.current);
                  if (tab === 'register' && value.trim().length >= 2 && /^[a-zA-Z0-9_.-]+$/.test(value.trim())) {
                    setUsernameStatus('checking');
                    checkTimer.current = setTimeout(() => { api.users.search(value.trim()).then(result => setUsernameStatus(result.some(user => user.username.toLowerCase() === value.trim().toLowerCase()) ? 'taken' : 'available')).catch(() => setUsernameStatus('idle')); }, 500);
                  }
                }} placeholder={t('design.usernamePlaceholder')} />
              {tab === 'register' && <p id="username-hint" className={`field-hint ${usernameStatus === 'taken' ? 'field-error' : ''}`} aria-live="polite">{t(`design.username_${usernameStatus}`)}</p>}
            </div>
            <div className="form-field">
              <label htmlFor="auth-password">{t('auth.password')}</label>
              <div className="password-field">
                <input id="auth-password" name="password" type={showPassword ? 'text' : 'password'} value={password} onChange={event => { setPassword(event.target.value); setError(null); }} autoComplete={tab === 'register' ? 'new-password' : 'current-password'} required minLength={tab === 'register' ? 8 : 1} placeholder={t('design.passwordPlaceholder')} aria-describedby={tab === 'register' ? 'password-hint' : undefined} />
                <button type="button" className="password-visibility" aria-label={t(showPassword ? 'design.hidePassword' : 'design.showPassword')} aria-pressed={showPassword} onClick={() => setShowPassword(value => !value)}>{showPassword ? <EyeOff size={18} /> : <Eye size={18} />}</button>
              </div>
              {tab === 'register' && <p id="password-hint" className="field-hint">{t('design.passwordHint')}</p>}
            </div>
            {tab === 'login' && mfaRequired && <div className="form-field">
              <label htmlFor="totp-code">{t('design.authenticatorCode')}</label>
              <input id="totp-code" className="otp-input" value={totpCode} onChange={event => setTotpCode(event.target.value.replace(/[^0-9]/g,'').slice(0,6))} inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" minLength={6} maxLength={6} required autoFocus />
            </div>}
            {error && <p className="form-feedback field-error" role="alert">{error}</p>}
            {!error && sessionNotice && <p className="form-feedback" role="status">{sessionNotice}</p>}
            <button type="submit" className="primary-button auth-submit" disabled={loading}>
              {loading ? <span className="button-spinner" aria-hidden="true" /> : null}
              {t(loading ? 'common.loading' : tab === 'login' ? 'auth.loginButton' : 'auth.registerButton')}
              {!loading && <ArrowRight size={17} aria-hidden="true" />}
            </button>
          </form>
        </div>
        <p className="auth-caption">{t('design.privateSpace')}</p>
      </section>
      <footer className="auth-footer"><span>{t('design.earlyAccess')}</span><span>{t('design.selfHosted')}</span></footer>
      {showServers && <ServerSettings onClose={() => setShowServers(false)} />}
    </main>
  );
}

async function registerE2EEDeviceBestEffort(userId: string): Promise<void> {
  try { await ensureLocalDeviceRegistered(userId); }
  catch (err) { console.warn('[E2EE] Device key registration postponed', err); }
}
