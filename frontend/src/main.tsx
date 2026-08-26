import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import { HomePage } from './pages/HomePage'
import { RegisterPage } from './pages/RegisterPage'
import { FormsAdminPage } from './pages/FormsAdminPage'
import { FormEditorPage } from './pages/FormEditorPage'
import { PublicFormPage } from './pages/PublicFormPage'
import { FormResponsesPage } from './pages/FormResponsesPage'

const queryClient = new QueryClient()
const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#f4b942',
      contrastText: '#17120a',
    },
    background: {
      default: '#090a0d',
      paper: '#121419',
    },
  },
  shape: {
    borderRadius: 14,
  },
  typography: {
    fontFamily:
      'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    button: {
      fontWeight: 700,
      textTransform: 'none',
    },
  },
})
const router = createBrowserRouter([
  { path: '/', element: <HomePage /> },
  { path: '/register', element: <RegisterPage /> },
  { path: '/forms/:slug', element: <PublicFormPage /> },
  { path: '/admin/forms', element: <FormsAdminPage /> },
  { path: '/admin/forms/new', element: <FormEditorPage /> },
  { path: '/admin/forms/:formId/responses', element: <FormResponsesPage /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
