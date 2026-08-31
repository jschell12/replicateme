import {useState, useEffect} from 'react'

import {GetConfig, SaveConfig, SelectFile, IsRAGAvailable, GetModels} from '../../wailsjs/go/main/App'
import {config as configModels} from '../../wailsjs/go/models'

interface QuirkToggles {
    misspellings?: boolean
    grammarErrors?: boolean
    missingApostrophes?: boolean
    lowercaseI?: boolean
    skipPunctuation?: boolean
    doubleSpaces?: boolean
    fragments?: boolean
}

interface RAGConfig {
    enabled: boolean
    qdrantUrl?: string
    ollamaUrl?: string
    embedModel?: string
}

interface Config {
    provider: string
    model?: string
    baseUrl?: string
    quirkLevel: number
    defaultPlatform: string
    persona?: string
    quirks?: QuirkToggles
    rag?: RAGConfig
}

const providers = ['anthropic', 'openai', 'claude-cli', 'ollama']
const platforms = ['imessage', 'slack', 'email', 'github', 'twitter', 'discord', 'reddit', 'instagram', 'tiktok']

// Fallback models if the API is unreachable
const fallbackModels: Record<string, string[]> = {
    anthropic: ['claude-sonnet-4-6-20250725', 'claude-opus-4-6-20250725', 'claude-haiku-4-5-20251001'],
    openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4.1', 'gpt-4.1-mini'],
    ollama: ['mistral-small:24b', 'qwen2.5:72b'],
    'claude-cli': [],
}
const quirkFields: {key: keyof QuirkToggles; label: string}[] = [
    {key: 'misspellings', label: 'Include common misspellings'},
    {key: 'grammarErrors', label: 'Include grammar shortcuts'},
    {key: 'missingApostrophes', label: 'Skip apostrophes (dont, cant, im)'},
    {key: 'lowercaseI', label: 'Use lowercase "i"'},
    {key: 'skipPunctuation', label: 'Drop punctuation'},
    {key: 'doubleSpaces', label: 'Allow double spaces'},
    {key: 'fragments', label: 'Use sentence fragments'},
]

export default function SettingsScreen() {
    const [config, setConfig] = useState<Config>({
        provider: 'anthropic',
        quirkLevel: 50,
        defaultPlatform: 'imessage',
        quirks: {},
        rag: {enabled: false},
    })
    const [saved, setSaved] = useState(false)
    const [error, setError] = useState('')
    const [ragAvailable, setRagAvailable] = useState(false)
    const [models, setModels] = useState<string[]>([])
    const [loadingModels, setLoadingModels] = useState(false)

    useEffect(() => {
        GetConfig().then(cfg => {
            setConfig({
                ...cfg,
                quirks: cfg.quirks || {},
                rag: cfg.rag || {enabled: false},
            })
        }).catch(() => {})

        IsRAGAvailable().then(setRagAvailable).catch(() => {})
    }, [])

    // fetch models when provider or baseURL changes
    useEffect(() => {
        if (config.provider === 'claude-cli') {
            setModels([])
            return
        }
        setLoadingModels(true)
        GetModels(config.provider, config.baseUrl || '')
            .then(m => setModels(m || []))
            .catch(() => setModels(fallbackModels[config.provider] || []))
            .finally(() => setLoadingModels(false))
    }, [config.provider, config.baseUrl])

    const handleSave = async () => {
        setSaved(false)
        setError('')
        try {
            await SaveConfig(configModels.Config.createFrom(config))
            setSaved(true)
            setTimeout(() => setSaved(false), 2000)
        } catch (e: any) {
            setError(e?.message || String(e))
        }
    }

    const handleBrowsePersona = async () => {
        try {
            const path = await SelectFile()
            if (path) setConfig(prev => ({...prev, persona: path}))
        } catch {}
    }

    const updateQuirk = (key: keyof QuirkToggles, value: boolean) => {
        setConfig(prev => ({
            ...prev,
            quirks: {...(prev.quirks || {}), [key]: value},
        }))
    }

    const updateRag = (updates: Partial<RAGConfig>) => {
        setConfig(prev => ({
            ...prev,
            rag: {...(prev.rag || {enabled: false}), ...updates},
        }))
    }

    const showBaseUrl = config.provider === 'openai' || config.provider === 'ollama'

    // Check env vars for API key status
    const apiKeyLabel = config.provider === 'anthropic'
        ? 'ANTHROPIC_API_KEY'
        : config.provider === 'openai'
        ? 'OPENAI_API_KEY'
        : null

    return (
        <div className="p-6 max-w-3xl mx-auto">
            <h2 className="text-xl font-semibold mb-5">Settings</h2>

            <div className="space-y-6">
                {/* Provider */}
                <section className="bg-gray-800 border border-gray-700 rounded-lg p-5">
                    <h3 className="text-sm font-medium text-gray-300 mb-3">LLM Provider</h3>
                    <div className="flex gap-3 flex-wrap">
                        {providers.map(p => (
                            <label key={p} className="flex items-center gap-2 cursor-pointer">
                                <input
                                    type="radio"
                                    name="provider"
                                    value={p}
                                    checked={config.provider === p}
                                    onChange={() => setConfig(prev => ({...prev, provider: p}))}
                                    className="accent-green-500"
                                />
                                <span className="text-sm text-gray-200">{p}</span>
                            </label>
                        ))}
                    </div>

                    <div className="mt-4 space-y-3">
                        {config.provider !== 'claude-cli' && (
                        <div>
                            <label className="block text-xs text-gray-400 mb-1">
                                Model {loadingModels && <span className="text-gray-500">(loading...)</span>}
                            </label>
                            <select
                                className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500"
                                value={config.model || ''}
                                onChange={e => setConfig(prev => ({...prev, model: e.target.value}))}
                            >
                                <option value="">Default</option>
                                {models.map(m => (
                                    <option key={m} value={m}>{m}</option>
                                ))}
                            </select>
                        </div>
                        )}

                        {showBaseUrl && (
                            <div>
                                <label className="block text-xs text-gray-400 mb-1">Base URL</label>
                                <input
                                    type="text"
                                    placeholder={config.provider === 'ollama' ? 'http://10.0.0.2:11434' : 'https://api.openai.com'}
                                    className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                    value={config.baseUrl || ''}
                                    onChange={e => setConfig(prev => ({...prev, baseUrl: e.target.value}))}
                                />
                            </div>
                        )}

                        {apiKeyLabel && (
                            <p className="text-xs text-gray-500">
                                API key is read from the <code className="text-gray-400">{apiKeyLabel}</code> environment variable.
                            </p>
                        )}
                    </div>
                </section>

                {/* Defaults */}
                <section className="bg-gray-800 border border-gray-700 rounded-lg p-5">
                    <h3 className="text-sm font-medium text-gray-300 mb-3">Defaults</h3>
                    <div className="space-y-3">
                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Tone</label>
                            <div className="flex items-center gap-3">
                                <span className="text-xs text-gray-500 w-12">Casual</span>
                                <input
                                    type="range"
                                    min="0"
                                    max="100"
                                    value={config.quirkLevel}
                                    onChange={e => setConfig(prev => ({...prev, quirkLevel: Number(e.target.value)}))}
                                    className="flex-1 accent-green-500"
                                />
                                <span className="text-xs text-gray-500 w-12 text-right">Formal</span>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Default Platform</label>
                            <select
                                className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500"
                                value={config.defaultPlatform}
                                onChange={e => setConfig(prev => ({...prev, defaultPlatform: e.target.value}))}
                            >
                                {platforms.map(p => (
                                    <option key={p} value={p}>{p}</option>
                                ))}
                            </select>
                        </div>

                        <div>
                            <label className="block text-xs text-gray-400 mb-1">Persona File</label>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={handleBrowsePersona}
                                    className="bg-gray-700 hover:bg-gray-600 text-gray-300 px-3 py-1.5 rounded text-sm transition-colors"
                                >
                                    Browse
                                </button>
                                <span className="text-sm text-gray-400 truncate flex-1">
                                    {config.persona || 'None'}
                                </span>
                                {config.persona && (
                                    <button
                                        onClick={() => setConfig(prev => ({...prev, persona: ''}))}
                                        className="text-xs text-gray-500 hover:text-gray-300"
                                    >
                                        Clear
                                    </button>
                                )}
                            </div>
                        </div>
                    </div>
                </section>

                {/* Quirk toggles */}
                <section className="bg-gray-800 border border-gray-700 rounded-lg p-5">
                    <h3 className="text-sm font-medium text-gray-300 mb-3">Writing Style</h3>
                    <div className="grid grid-cols-2 gap-2">
                        {quirkFields.map(({key, label}) => (
                            <label key={key} className="flex items-center gap-2 cursor-pointer py-1">
                                <input
                                    type="checkbox"
                                    checked={(config.quirks as any)?.[key] === true}
                                    onChange={e => updateQuirk(key, e.target.checked)}
                                    className="accent-green-500 rounded"
                                />
                                <span className="text-sm text-gray-300">{label}</span>
                            </label>
                        ))}
                    </div>
                </section>

                {/* RAG */}
                <section className="bg-gray-800 border border-gray-700 rounded-lg p-5">
                    <div className="flex items-center justify-between mb-3">
                        <h3 className="text-sm font-medium text-gray-300">RAG (Vector Search)</h3>
                        <div className="flex items-center gap-2">
                            {config.rag?.enabled && (
                                <span className={`w-2 h-2 rounded-full ${ragAvailable ? 'bg-green-400' : 'bg-red-400'}`} />
                            )}
                            <label className="flex items-center gap-2 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={config.rag?.enabled || false}
                                    onChange={e => updateRag({enabled: e.target.checked})}
                                    className="accent-green-500"
                                />
                                <span className="text-xs text-gray-400">Enable</span>
                            </label>
                        </div>
                    </div>

                    {config.rag?.enabled && (
                        <div className="space-y-3">
                            <div>
                                <label className="block text-xs text-gray-400 mb-1">Qdrant URL</label>
                                <input
                                    type="text"
                                    placeholder="http://10.0.0.2:6333"
                                    className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                    value={config.rag.qdrantUrl || ''}
                                    onChange={e => updateRag({qdrantUrl: e.target.value})}
                                />
                            </div>
                            <div>
                                <label className="block text-xs text-gray-400 mb-1">Ollama URL</label>
                                <input
                                    type="text"
                                    placeholder="http://10.0.0.2:11434"
                                    className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                    value={config.rag.ollamaUrl || ''}
                                    onChange={e => updateRag({ollamaUrl: e.target.value})}
                                />
                            </div>
                            <div>
                                <label className="block text-xs text-gray-400 mb-1">Embed Model</label>
                                <input
                                    type="text"
                                    placeholder="bge-m3"
                                    className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                    value={config.rag.embedModel || ''}
                                    onChange={e => updateRag({embedModel: e.target.value})}
                                />
                            </div>
                        </div>
                    )}
                </section>

                {/* Save */}
                <div className="flex items-center gap-3">
                    <button
                        onClick={handleSave}
                        className="bg-green-600 hover:bg-green-500 text-white px-6 py-2.5 rounded-lg text-sm font-medium transition-colors"
                    >
                        Save
                    </button>
                    {saved && <span className="text-sm text-green-400">Saved</span>}
                    {error && <span className="text-sm text-red-400">{error}</span>}
                </div>
            </div>
        </div>
    )
}
