import {useState, useEffect} from 'react'

import {GetAllProfiles, GetCorpusStats} from '../../wailsjs/go/main/App'

interface StyleProfile {
    averageLength: number
    capitalizesFirstWord: number
    usesContractions: number
    usesPeriods: number
    usesCommas: number
    usesExclamation: number
    usesQuestionMark: number
    usesEmoji: number
    commonPhrases: string[]
    typicalErrors: {pattern: string; frequency: number; examples: string[]}[]
    sentenceFragmentRatio: number
    lowercaseIRatio: number
}

interface PlatformProfile {
    Platform: string
    Profile: StyleProfile
    MessageCount: number
}

interface CorpusStats {
    totalMessages: number
    byPlatform: {platform: string; count: number}[]
    profiles: {platform: string; messageCount: number; updatedAt: string}[]
}

function pct(ratio: number): string {
    return `${Math.round(ratio * 100)}%`
}

export default function ProfileScreen() {
    const [profiles, setProfiles] = useState<PlatformProfile[]>([])
    const [stats, setStats] = useState<CorpusStats | null>(null)
    const [activeTab, setActiveTab] = useState('')
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        Promise.all([
            GetAllProfiles(),
            GetCorpusStats(),
        ]).then(([profs, st]) => {
            setProfiles(profs || [])
            setStats(st)
            if (profs && profs.length > 0) {
                const combined = profs.find((p: PlatformProfile) => p.Platform === 'combined')
                setActiveTab(combined ? 'combined' : profs[0].Platform)
            }
        }).catch(() => {}).finally(() => setLoading(false))
    }, [])

    if (loading) {
        return (
            <div className="p-6 flex items-center justify-center text-gray-400">
                <div className="w-5 h-5 border-2 border-green-500 border-t-transparent rounded-full animate-spin mr-2" />
                Loading profiles...
            </div>
        )
    }

    if (profiles.length === 0) {
        return (
            <div className="p-6">
                <h2 className="text-xl font-semibold mb-5">Profile</h2>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
                    No profiles yet. Ingest some messages to build your style profile.
                </div>
            </div>
        )
    }

    const active = profiles.find(p => p.Platform === activeTab)
    const prof = active?.Profile

    return (
        <div className="p-6 max-w-4xl mx-auto">
            <h2 className="text-xl font-semibold mb-5">Profile</h2>

            {/* Platform tabs */}
            <div className="flex gap-1 border-b border-gray-700 mb-6">
                {profiles.map(p => (
                    <button
                        key={p.Platform}
                        onClick={() => setActiveTab(p.Platform)}
                        className={`px-4 py-2 text-sm transition-colors border-b-2 ${
                            activeTab === p.Platform
                                ? 'border-green-400 text-green-400'
                                : 'border-transparent text-gray-400 hover:text-gray-200'
                        }`}
                    >
                        {p.Platform}
                    </button>
                ))}
            </div>

            {prof && (
                <>
                    {/* Stats grid */}
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
                        <StatCard label="Avg Length" value={`${prof.averageLength} chars`} />
                        <StatCard label="Capitalizes" value={pct(prof.capitalizesFirstWord)} />
                        <StatCard label="Periods" value={pct(prof.usesPeriods)} />
                        <StatCard label="Question Marks" value={pct(prof.usesQuestionMark)} />
                        <StatCard label="Emoji" value={pct(prof.usesEmoji)} />
                        <StatCard label="Fragments" value={pct(prof.sentenceFragmentRatio)} />
                        <StatCard label="Lowercase i" value={pct(prof.lowercaseIRatio)} />
                        <StatCard label="Messages" value={String(active?.MessageCount || 0)} />
                    </div>

                    {/* Quirks */}
                    {prof.typicalErrors && prof.typicalErrors.length > 0 && (
                        <div className="mb-6">
                            <h3 className="text-sm font-medium text-gray-400 mb-3">Quirks</h3>
                            <div className="bg-gray-800 border border-gray-700 rounded-lg divide-y divide-gray-700">
                                {prof.typicalErrors.map((err, i) => (
                                    <div key={i} className="px-4 py-3 flex items-center justify-between">
                                        <span className="text-sm text-gray-200">{err.pattern}</span>
                                        <span className="text-xs text-gray-400 bg-gray-700 px-2 py-1 rounded">{err.frequency}x</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Common phrases */}
                    {prof.commonPhrases && prof.commonPhrases.length > 0 && (
                        <div className="mb-6">
                            <h3 className="text-sm font-medium text-gray-400 mb-3">Common Phrases</h3>
                            <div className="flex flex-wrap gap-2">
                                {prof.commonPhrases.slice(0, 15).map((phrase, i) => (
                                    <span key={i} className="bg-gray-800 border border-gray-700 px-3 py-1.5 rounded text-xs text-gray-300">
                                        "{phrase}"
                                    </span>
                                ))}
                            </div>
                        </div>
                    )}
                </>
            )}

            {/* Corpus stats */}
            {stats && (
                <div className="mt-8 border-t border-gray-700 pt-6">
                    <h3 className="text-sm font-medium text-gray-400 mb-3">Corpus Summary</h3>
                    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                        <p className="text-sm text-gray-300 mb-3">Total messages: <span className="text-green-400 font-medium">{stats.totalMessages}</span></p>
                        {stats.byPlatform && stats.byPlatform.length > 0 && (
                            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                                {stats.byPlatform.map(bp => (
                                    <div key={bp.platform} className="flex items-center justify-between bg-gray-900 rounded px-3 py-2">
                                        <span className="text-xs text-gray-400">{bp.platform}</span>
                                        <span className="text-xs text-gray-200">{bp.count}</span>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    )
}

function StatCard({label, value}: {label: string; value: string}) {
    return (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-3 text-center">
            <p className="text-xs text-gray-400 mb-1">{label}</p>
            <p className="text-lg font-semibold text-gray-100">{value}</p>
        </div>
    )
}
