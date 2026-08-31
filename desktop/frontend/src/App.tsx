import {Routes, Route, NavLink, Navigate} from 'react-router-dom'
import GenerateScreen from './screens/Generate'
import ProfileScreen from './screens/Profile'
import IngestScreen from './screens/Ingest'
import SettingsScreen from './screens/Settings'

const navItems = [
    {path: '/generate', label: 'Generate', icon: '⚡'},
    {path: '/profile', label: 'Profile', icon: '📊'},
    {path: '/ingest', label: 'Ingest', icon: '📥'},
    {path: '/settings', label: 'Settings', icon: '⚙️'},
]

function App() {
    return (
        <div className="flex h-screen bg-gray-900 text-gray-100">
            <nav className="w-52 bg-gray-800 border-r border-gray-700 flex flex-col">
                <div className="px-5 py-5 border-b border-gray-700">
                    <h1 className="text-lg font-semibold text-green-400 tracking-tight">replicateme</h1>
                </div>
                <div className="flex-1 py-3">
                    {navItems.map(item => (
                        <NavLink
                            key={item.path}
                            to={item.path}
                            className={({isActive}) =>
                                `flex items-center gap-3 px-5 py-2.5 text-sm transition-colors ${
                                    isActive
                                        ? 'bg-gray-700 text-green-400 border-r-2 border-green-400'
                                        : 'text-gray-400 hover:text-gray-200 hover:bg-gray-750'
                                }`
                            }
                        >
                            <span>{item.icon}</span>
                            <span>{item.label}</span>
                        </NavLink>
                    ))}
                </div>
            </nav>
            <main className="flex-1 overflow-y-auto">
                <Routes>
                    <Route path="/generate" element={<GenerateScreen />} />
                    <Route path="/profile" element={<ProfileScreen />} />
                    <Route path="/ingest" element={<IngestScreen />} />
                    <Route path="/settings" element={<SettingsScreen />} />
                    <Route path="*" element={<Navigate to="/generate" replace />} />
                </Routes>
            </main>
        </div>
    )
}

export default App
