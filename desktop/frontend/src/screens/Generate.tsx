import {useState, useEffect} from 'react'

// Wails bindings will be generated - declare types
declare function Generate(platform: string, context: string, quirkLevel: number, personaPath: string, variants: number): Promise<string[]>
declare function CopyToClipboard(text: string): Promise<void>
declare function GetConfig(): Promise<any>

// Import from wailsjs
import {Generate as WailsGenerate, CopyToClipboard as WailsCopy, GetConfig as WailsGetConfig} from '../../wailsjs/go/main/App'

const platforms = ['imessage', 'slack', 'email', 'github', 'twitter', 'discord', 'reddit', 'instagram', 'tiktok']

export default function GenerateScreen() {
    const [contextText, setContextText] = useState('')
    const [platform, setPlatform] = useState('imessage')
    const [quirkLevel, setQuirkLevel] = useState(50)
    const [variants, setVariants] = useState(3)
    const [results, setResults] = useState<string[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')
    const [persona, setPersona] = useState('')
    const [copiedIdx, setCopiedIdx] = useState<number | null>(null)

    useEffect(() => {
        WailsGetConfig().then(cfg => {
            if (cfg.defaultPlatform) setPlatform(cfg.defaultPlatform)
            if (cfg.quirkLevel !== undefined) setQuirkLevel(cfg.quirkLevel)
            if (cfg.persona) setPersona(cfg.persona)
        }).catch(() => {})
    }, [])

    const handleGenerate = async () => {
        if (!contextText.trim()) return
        setLoading(true)
        setError('')
        setResults([])
        try {
            const res = await WailsGenerate(platform, contextText, quirkLevel, persona, variants)
            setResults(res || [])
        } catch (e: any) {
            setError(e?.message || String(e))
        } finally {
            setLoading(false)
        }
    }

    const handleCopy = async (text: string, idx: number) => {
        try {
            await WailsCopy(text)
            setCopiedIdx(idx)
            setTimeout(() => setCopiedIdx(null), 1500)
        } catch {}
    }

    return (
        <div className="p-6 max-w-4xl mx-auto">
            <h2 className="text-xl font-semibold mb-5">Generate</h2>

            <textarea
                className="w-full h-36 bg-gray-800 border border-gray-700 rounded-lg p-4 text-gray-100 text-sm resize-none focus:outline-none focus:border-green-500 placeholder-gray-500"
                placeholder="What context or prompt do you want to generate a response for?"
                value={contextText}
                onChange={e => setContextText(e.target.value)}
            />

            <div className="flex items-center gap-4 mt-4 flex-wrap">
                <div className="flex items-center gap-2">
                    <label className="text-sm text-gray-400">Platform</label>
                    <select
                        className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm text-gray-100 focus:outline-none focus:border-green-500"
                        value={platform}
                        onChange={e => setPlatform(e.target.value)}
                    >
                        {platforms.map(p => (
                            <option key={p} value={p}>{p}</option>
                        ))}
                    </select>
                </div>

                <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500">Casual</span>
                    <input
                        type="range"
                        min="0"
                        max="100"
                        value={quirkLevel}
                        onChange={e => setQuirkLevel(Number(e.target.value))}
                        className="w-28 accent-green-500"
                    />
                    <span className="text-xs text-gray-500">Formal</span>
                </div>

                <div className="flex items-center gap-2">
                    <label className="text-sm text-gray-400">Variants</label>
                    <input
                        type="number"
                        min="1"
                        max="10"
                        value={variants}
                        onChange={e => setVariants(Number(e.target.value))}
                        className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-sm text-gray-100 w-16 focus:outline-none focus:border-green-500"
                    />
                </div>

                <button
                    onClick={handleGenerate}
                    disabled={loading || !contextText.trim()}
                    className="ml-auto bg-green-600 hover:bg-green-500 disabled:bg-gray-700 disabled:text-gray-500 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors"
                >
                    {loading ? 'Generating...' : 'Generate'}
                </button>
            </div>

            {persona && (
                <div className="mt-3">
                    <span className="inline-block bg-gray-700 text-green-400 text-xs px-2 py-1 rounded">
                        Persona: {persona.split('/').pop()}
                    </span>
                </div>
            )}

            {error && (
                <div className="mt-4 bg-red-900/50 border border-red-700 rounded-lg p-3 text-red-300 text-sm">
                    {error}
                </div>
            )}

            {loading && (
                <div className="mt-6 flex items-center justify-center gap-2 text-gray-400">
                    <div className="w-5 h-5 border-2 border-green-500 border-t-transparent rounded-full animate-spin" />
                    <span className="text-sm">Generating variants...</span>
                </div>
            )}

            {results.length > 0 && (
                <div className="mt-6 space-y-3">
                    {results.map((text, i) => (
                        <div key={i} className="bg-gray-800 border border-gray-700 rounded-lg p-4 group">
                            <div className="flex items-start justify-between gap-3">
                                <p className="text-sm text-gray-200 whitespace-pre-wrap flex-1">{text}</p>
                                <button
                                    onClick={() => handleCopy(text, i)}
                                    className="shrink-0 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 px-3 py-1.5 rounded transition-colors"
                                >
                                    {copiedIdx === i ? 'Copied!' : 'Copy'}
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}
