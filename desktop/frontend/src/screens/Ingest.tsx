import {useState, useEffect} from 'react'

import {IngestSource, GetSources, SelectFile, GetCorpusStats} from '../../wailsjs/go/main/App'

interface SourceInfo {
    name: string
    description: string
    requiresFile: boolean
}

interface IngestResult {
    messageCount: number
    newCount: number
    profileNote: string
}

interface CorpusStats {
    totalMessages: number
    byPlatform: {platform: string; count: number}[]
    profiles: {platform: string; messageCount: number; updatedAt: string}[]
}

export default function IngestScreen() {
    const [sources, setSources] = useState<SourceInfo[]>([])
    const [selectedSource, setSelectedSource] = useState('')
    const [filePath, setFilePath] = useState('')
    const [options, setOptions] = useState<Record<string, string>>({})
    const [loading, setLoading] = useState(false)
    const [result, setResult] = useState<IngestResult | null>(null)
    const [error, setError] = useState('')
    const [stats, setStats] = useState<CorpusStats | null>(null)

    useEffect(() => {
        GetSources().then(s => {
            setSources(s || [])
            if (s && s.length > 0) setSelectedSource(s[0].name)
        }).catch(() => {})

        GetCorpusStats().then(s => setStats(s)).catch(() => {})
    }, [])

    const currentSource = sources.find(s => s.name === selectedSource)

    const handleSelectFile = async () => {
        try {
            const path = await SelectFile()
            if (path) setFilePath(path)
        } catch {}
    }

    const handleIngest = async () => {
        setLoading(true)
        setError('')
        setResult(null)
        try {
            const res = await IngestSource(selectedSource, filePath, options)
            setResult(res)
            // Refresh stats
            GetCorpusStats().then(s => setStats(s)).catch(() => {})
        } catch (e: any) {
            setError(e?.message || String(e))
        } finally {
            setLoading(false)
        }
    }

    const updateOption = (key: string, value: string) => {
        setOptions(prev => ({...prev, [key]: value}))
    }

    const handleSourceChange = (name: string) => {
        setSelectedSource(name)
        setFilePath('')
        setOptions({})
        setResult(null)
        setError('')
    }

    return (
        <div className="p-6 max-w-4xl mx-auto">
            <h2 className="text-xl font-semibold mb-5">Ingest</h2>

            <div className="bg-gray-800 border border-gray-700 rounded-lg p-5 space-y-4">
                {/* Source selection */}
                <div>
                    <label className="block text-sm text-gray-400 mb-1.5">Source</label>
                    <select
                        className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500"
                        value={selectedSource}
                        onChange={e => handleSourceChange(e.target.value)}
                    >
                        {sources.map(s => (
                            <option key={s.name} value={s.name}>{s.name} - {s.description}</option>
                        ))}
                    </select>
                </div>

                {/* File picker (when needed) */}
                {currentSource?.requiresFile && (
                    <div>
                        <label className="block text-sm text-gray-400 mb-1.5">File</label>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={handleSelectFile}
                                className="bg-gray-700 hover:bg-gray-600 text-gray-300 px-4 py-2 rounded text-sm transition-colors"
                            >
                                Browse...
                            </button>
                            <span className="text-sm text-gray-400 truncate flex-1">
                                {filePath || 'No file selected'}
                            </span>
                        </div>
                    </div>
                )}

                {/* Conditional fields based on source */}
                {selectedSource === 'slack' && (
                    <div>
                        <label className="block text-sm text-gray-400 mb-1.5">User Name</label>
                        <input
                            type="text"
                            placeholder="Your display name in Slack"
                            className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                            value={options.userName || ''}
                            onChange={e => updateOption('userName', e.target.value)}
                        />
                    </div>
                )}

                {selectedSource === 'gmail' && (
                    <div>
                        <label className="block text-sm text-gray-400 mb-1.5">Email Address</label>
                        <input
                            type="email"
                            placeholder="your@email.com"
                            className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                            value={options.email || ''}
                            onChange={e => updateOption('email', e.target.value)}
                        />
                    </div>
                )}

                {selectedSource === 'instagram' && (
                    <div>
                        <label className="block text-sm text-gray-400 mb-1.5">Username</label>
                        <input
                            type="text"
                            placeholder="Your Instagram username"
                            className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                            value={options.username || ''}
                            onChange={e => updateOption('username', e.target.value)}
                        />
                    </div>
                )}

                {selectedSource === 'github' && (
                    <>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1.5">Repos (comma-separated paths)</label>
                            <input
                                type="text"
                                placeholder="/path/to/repo1, /path/to/repo2"
                                className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                value={options.repos || ''}
                                onChange={e => updateOption('repos', e.target.value)}
                            />
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-1.5">Email (git author filter)</label>
                            <input
                                type="email"
                                placeholder="your@email.com"
                                className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-green-500 placeholder-gray-600"
                                value={options.email || ''}
                                onChange={e => updateOption('email', e.target.value)}
                            />
                        </div>
                    </>
                )}

                {/* Ingest button */}
                <button
                    onClick={handleIngest}
                    disabled={loading || (currentSource?.requiresFile && !filePath)}
                    className="w-full bg-green-600 hover:bg-green-500 disabled:bg-gray-700 disabled:text-gray-500 text-white py-2.5 rounded-lg text-sm font-medium transition-colors"
                >
                    {loading ? (
                        <span className="flex items-center justify-center gap-2">
                            <span className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                            Ingesting...
                        </span>
                    ) : 'Ingest'}
                </button>
            </div>

            {/* Result */}
            {error && (
                <div className="mt-4 bg-red-900/50 border border-red-700 rounded-lg p-3 text-red-300 text-sm">
                    {error}
                </div>
            )}

            {result && (
                <div className="mt-4 bg-green-900/30 border border-green-700 rounded-lg p-4">
                    <p className="text-sm text-green-300">
                        Found {result.messageCount} messages, {result.newCount} new.
                    </p>
                    {result.profileNote && (
                        <p className="text-xs text-green-400 mt-1">{result.profileNote}</p>
                    )}
                </div>
            )}

            {/* Past ingests / corpus stats */}
            {stats && stats.byPlatform && stats.byPlatform.length > 0 && (
                <div className="mt-6">
                    <h3 className="text-sm font-medium text-gray-400 mb-3">Corpus</h3>
                    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                        <p className="text-sm text-gray-300 mb-3">Total: <span className="text-green-400">{stats.totalMessages}</span> messages</p>
                        <div className="space-y-1.5">
                            {stats.byPlatform.map(bp => (
                                <div key={bp.platform} className="flex items-center justify-between">
                                    <span className="text-xs text-gray-400">{bp.platform}</span>
                                    <div className="flex items-center gap-2 flex-1 mx-3">
                                        <div className="flex-1 bg-gray-900 rounded-full h-1.5">
                                            <div
                                                className="bg-green-500 h-1.5 rounded-full"
                                                style={{width: `${Math.min(100, (bp.count / stats.totalMessages) * 100)}%`}}
                                            />
                                        </div>
                                    </div>
                                    <span className="text-xs text-gray-300 w-12 text-right">{bp.count}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}
