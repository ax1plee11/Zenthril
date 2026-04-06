/**
 * GifPicker — поиск GIF через несколько источников + ручной ввод URL
 */
import { useState, useEffect, useRef, useCallback } from "react";

interface GifPickerProps {
  onSelect: (url: string) => void;
  onClose: () => void;
}

// GIF источники. Ключи должны приходить из env (VITE_*), иначе остаётся вкладка Paste URL.
const CATEGORIES = ["trending","meme","anime","funny","reaction","love","gaming","cat","dog","fire","wow"];

interface GifItem { id: string; url: string; preview: string; }

const TENOR_KEY = (import.meta as any).env?.VITE_TENOR_KEY as string | undefined;
const GIPHY_KEY = (import.meta as any).env?.VITE_GIPHY_KEY as string | undefined;

async function normalizeGifUrl(input: string): Promise<string> {
  const url = input.trim();
  if (!url) return url;

  // Tenor share page: ...-<id>
  if (/^https?:\/\/(www\.)?tenor\.com\/view\//i.test(url)) {
    if (!TENOR_KEY) return url;
    const m = url.match(/-([0-9]+)(?:\?.*)?$/);
    const id = m?.[1];
    if (id) {
      const res = await fetch(
        `https://tenor.googleapis.com/v2/posts?ids=${encodeURIComponent(id)}&key=${encodeURIComponent(TENOR_KEY)}&media_filter=gif,mediumgif,tinygif`,
      );
      const data = (await res.json()) as { results?: any[] };
      const g = (data.results ?? [])[0];
      const fmt = g?.media_formats ?? {};
      const direct = fmt.gif?.url || fmt.mediumgif?.url || fmt.tinygif?.url;
      if (direct) return direct;
    }
  }

  // Giphy share page: ...-<id> or /gifs/<slug>
  if (/^https?:\/\/(www\.)?giphy\.com\/gifs\//i.test(url)) {
    if (!GIPHY_KEY) return url;
    const m = url.match(/-([a-zA-Z0-9]+)(?:\?.*)?$/);
    const id = m?.[1];
    if (id) {
      const res = await fetch(`https://api.giphy.com/v1/gifs/${encodeURIComponent(id)}?api_key=${encodeURIComponent(GIPHY_KEY)}`);
      const data = await res.json();
      const direct = data?.data?.images?.original?.url ?? data?.data?.images?.downsized_large?.url;
      if (direct) return direct;
    }
  }

  return url;
}

async function fetchTenor(q: string): Promise<GifItem[]> {
  if (!TENOR_KEY) return [];
  const endpoint = q
    ? `https://tenor.googleapis.com/v2/search?q=${encodeURIComponent(q)}&key=${encodeURIComponent(TENOR_KEY)}&limit=20&media_filter=gif,mediumgif,tinygif`
    : `https://tenor.googleapis.com/v2/featured?key=${encodeURIComponent(TENOR_KEY)}&limit=20&media_filter=gif,mediumgif,tinygif`;
  const res  = await fetch(endpoint);
  const data = await res.json() as { results?: any[] };
  return (data.results ?? []).map((g: any) => {
    const fmt = g.media_formats ?? {};
    // Приоритет качества: gif > mediumgif > tinygif
    const url     = fmt.gif?.url || fmt.mediumgif?.url || fmt.tinygif?.url || "";
    const preview = fmt.tinygif?.url || fmt.mediumgif?.url || fmt.gif?.url || "";
    return { id: String(g.id), url, preview };
  }).filter((g: GifItem) => g.url);
}

async function fetchGiphy(q: string): Promise<GifItem[]> {
  if (!GIPHY_KEY) return [];
  const endpoint = q
    ? `https://api.giphy.com/v1/gifs/search?api_key=${encodeURIComponent(GIPHY_KEY)}&q=${encodeURIComponent(q)}&limit=20&rating=g`
    : `https://api.giphy.com/v1/gifs/trending?api_key=${encodeURIComponent(GIPHY_KEY)}&limit=20&rating=g`;
  const res  = await fetch(endpoint);
  const data = await res.json();
  return (data.data ?? []).map((g: any) => ({
    id: g.id,
    // original — полное качество, без даунскейла
    url: g.images?.original?.url ?? g.images?.downsized_large?.url ?? "",
    preview: g.images?.fixed_height_small?.url ?? g.images?.downsized_small?.url ?? g.images?.original?.url ?? "",
  })).filter((g: GifItem) => g.url);
}

export default function GifPicker({ onSelect, onClose }: GifPickerProps) {
  const [query, setQuery]       = useState("");
  const [gifs, setGifs]         = useState<GifItem[]>([]);
  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState(false);
  const [category, setCategory] = useState("trending");
  const [urlInput, setUrlInput] = useState("");
  const [tab, setTab]           = useState<"search" | "url">("search");
  const inputRef  = useRef<HTMLInputElement>(null);
  const debounce  = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchGifs = useCallback(async (q: string) => {
    setLoading(true);
    setError(false);
    try {
      let items: GifItem[] = [];

      // Пробуем Tenor
      try {
        items = await fetchTenor(q);
      } catch (e) {
        console.warn("Tenor failed:", e);
      }

      // Если Tenor не дал результатов — пробуем Giphy
      if (items.length === 0) {
        try {
          items = await fetchGiphy(q);
        } catch (e) {
          console.warn("Giphy failed:", e);
        }
      }

      setGifs(items);
      if (items.length === 0) setError(true);
    } catch (e) {
      console.error("GIF fetch error:", e);
      setError(true);
      setGifs([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (tab === "search") fetchGifs(category === "trending" ? "" : category);
    inputRef.current?.focus();
  }, [category, tab]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSearch = useCallback((val: string) => {
    setQuery(val);
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = setTimeout(() => fetchGifs(val || (category === "trending" ? "" : category)), 500);
  }, [category, fetchGifs]);

  const handleUrlSend = useCallback(() => {
    void (async () => {
      const url = urlInput.trim();
      if (!url) return;
      try {
        const normalized = await normalizeGifUrl(url);
        onSelect(normalized);
      } catch {
        onSelect(url);
      } finally {
        onClose();
      }
    })();
  }, [urlInput, onSelect, onClose]);

  return (
    <div style={{
      position: "absolute" as const, bottom: "calc(100% + 8px)", left: 0, right: 0,
      background: "var(--bg-elevated)", border: "1px solid var(--border)",
      borderRadius: 16, boxShadow: "0 -4px 32px rgba(0,0,0,0.5)",
      zIndex: 200, overflow: "hidden", animation: "fadeUp 0.15s ease",
    }}>
      {/* Tabs */}
      <div style={{
        display: "flex", borderBottom: "1px solid var(--border)",
        padding: "8px 12px 0",
      }}>
        {[
          { id: "search", label: "🔍 Search GIFs" },
          { id: "url",    label: "🔗 Paste URL" },
        ].map(t => (
          <button key={t.id} onClick={() => setTab(t.id as "search" | "url")} style={{
            padding: "6px 14px", background: "none", border: "none",
            cursor: "pointer", fontSize: 12, fontWeight: 600,
            color: tab === t.id ? "var(--text-primary)" : "var(--text-muted)",
            borderBottom: `2px solid ${tab === t.id ? "var(--accent)" : "transparent"}`,
            marginBottom: -1, transition: "all 0.15s",
          }}>{t.label}</button>
        ))}
        <button onClick={onClose} style={{
          marginLeft: "auto", background: "none", border: "none",
          cursor: "pointer", color: "var(--text-muted)", fontSize: 16, padding: "4px 8px",
        }}>✕</button>
      </div>

      {/* URL tab */}
      {tab === "url" && (
        <div style={{ padding: 16, display: "flex", flexDirection: "column" as const, gap: 12 }}>
          <div style={{ fontSize: 12, color: "var(--text-muted)" }}>
            Paste any image or GIF URL — it will be displayed in the chat
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <input
              autoFocus
              value={urlInput}
              onChange={e => setUrlInput(e.target.value)}
              onKeyDown={e => e.key === "Enter" && handleUrlSend()}
              placeholder="https://example.com/image.gif"
              style={{
                flex: 1, padding: "10px 12px",
                background: "var(--bg-input)", border: "1px solid var(--border)",
                borderRadius: 8, color: "var(--text-primary)",
                fontSize: 13, outline: "none", fontFamily: "inherit",
              }}
            />
            <button onClick={handleUrlSend} disabled={!urlInput.trim()} style={{
              padding: "10px 16px",
              background: urlInput.trim() ? "linear-gradient(135deg, #7c6af7, #a78bfa)" : "var(--bg-input)",
              border: "none", borderRadius: 8, color: "#fff",
              cursor: urlInput.trim() ? "pointer" : "default",
              fontSize: 13, fontWeight: 600,
            }}>Send</button>
          </div>
          {/* Preview */}
          {urlInput.trim() && (
            <div style={{ borderRadius: 10, overflow: "hidden", border: "1px solid var(--border)", maxHeight: 200 }}>
              <img
                src={urlInput.trim()}
                alt="preview"
                style={{ width: "100%", maxHeight: 200, objectFit: "contain", display: "block", background: "var(--bg-input)" }}
                onError={e => { (e.target as HTMLImageElement).style.display = "none"; }}
              />
            </div>
          )}
        </div>
      )}

      {/* Search tab */}
      {tab === "search" && (
        <>
          <div style={{ padding: "10px 12px 8px" }}>
            {/* Search input */}
            <div style={{
              display: "flex", alignItems: "center", gap: 8,
              background: "var(--bg-input)", borderRadius: 8,
              padding: "7px 10px", border: "1px solid var(--border)", marginBottom: 8,
            }}>
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2">
                <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
              <input
                ref={inputRef}
                value={query}
                onChange={e => handleSearch(e.target.value)}
                placeholder="Search GIFs..."
                style={{
                  flex: 1, background: "none", border: "none", outline: "none",
                  color: "var(--text-primary)", fontSize: 13, fontFamily: "inherit",
                }}
              />
              {query && (
                <button onClick={() => { setQuery(""); fetchGifs(""); }} style={{
                  background: "none", border: "none", cursor: "pointer",
                  color: "var(--text-muted)", fontSize: 14, padding: 0,
                }}>✕</button>
              )}
            </div>

            {/* Categories */}
            {!query && (
              <div style={{ display: "flex", gap: 5, overflowX: "auto" as const, paddingBottom: 2 }}>
                {CATEGORIES.map(c => (
                  <button key={c} onClick={() => setCategory(c)} style={{
                    padding: "3px 10px", borderRadius: 20, border: "none",
                    background: category === c ? "var(--accent)" : "var(--bg-input)",
                    color: category === c ? "#fff" : "var(--text-muted)",
                    fontSize: 11, fontWeight: 600, cursor: "pointer",
                    whiteSpace: "nowrap" as const, transition: "all 0.15s",
                    textTransform: "capitalize" as const, flexShrink: 0,
                  }}>{c}</button>
                ))}
              </div>
            )}
          </div>

          {/* Grid */}
          <div style={{
            height: 260, overflowY: "auto" as const, padding: "0 8px 8px",
            display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 5,
            alignContent: "start",
          }}>
            {loading && (
              <div style={{
                gridColumn: "1/-1", display: "flex", alignItems: "center",
                justifyContent: "center", height: 200, gap: 10,
                color: "var(--text-muted)", fontSize: 13,
              }}>
                <div style={{
                  width: 18, height: 18, border: "2px solid var(--border)",
                  borderTopColor: "var(--accent)", borderRadius: "50%",
                  animation: "spin 0.7s linear infinite",
                }} />
                Loading...
              </div>
            )}

            {!loading && error && (
              <div style={{
                gridColumn: "1/-1", display: "flex", flexDirection: "column" as const,
                alignItems: "center", justifyContent: "center", height: 200,
                color: "var(--text-muted)", fontSize: 13, gap: 8,
              }}>
                <span style={{ fontSize: 28 }}>😕</span>
                <div>Could not load GIFs</div>
                <div style={{ fontSize: 11 }}>Use the "Paste URL" tab instead</div>
              </div>
            )}

            {!loading && !error && gifs.length === 0 && (
              <div style={{
                gridColumn: "1/-1", display: "flex", flexDirection: "column" as const,
                alignItems: "center", justifyContent: "center", height: 200,
                color: "var(--text-muted)", fontSize: 13, gap: 8,
              }}>
                <span style={{ fontSize: 28 }}>🔍</span>
                No GIFs found
              </div>
            )}

            {!loading && gifs.map(gif => (
              <div
                key={gif.id}
                onClick={() => { onSelect(gif.url); onClose(); }}
                style={{
                  borderRadius: 8, overflow: "hidden", cursor: "pointer",
                  background: "var(--bg-input)", aspectRatio: "16/9",
                  border: "1px solid var(--border)", transition: "transform 0.1s",
                }}
                onMouseEnter={e => { (e.currentTarget as HTMLDivElement).style.transform = "scale(1.04)"; }}
                onMouseLeave={e => { (e.currentTarget as HTMLDivElement).style.transform = "scale(1)"; }}
              >
                <img
                  src={gif.preview || gif.url}
                  alt="gif"
                  loading="lazy"
                  style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }}
                  onError={e => { (e.target as HTMLImageElement).style.opacity = "0.3"; }}
                />
              </div>
            ))}
          </div>

          <div style={{
            padding: "5px 12px", borderTop: "1px solid var(--border)",
            fontSize: 9, color: "var(--text-muted)", textAlign: "center" as const,
          }}>
            Powered by Tenor & Giphy
          </div>
        </>
      )}
    </div>
  );
}
