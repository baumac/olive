import { createRouter, createWebHistory } from 'vue-router'
import ErrorPage from '@/components/app/ErrorPage.vue'
import { refreshSession, signInPath } from '@/composables/useAuth'
import { tellPendingApproval } from '@/lib/approval'
import { pendingApproval, signedIn } from '@/lib/session'
import { safeRedirect } from '@/lib/validation'

declare module 'vue-router' {
  interface RouteMeta {
    // requiresAuth sends a signed out person to the sign in page first.
    requiresAuth?: boolean
    // requiresApproval keeps a user who waits for approval off the page.
    requiresApproval?: boolean
    // guestOnly sends a signed in person to the redirect path.
    guestOnly?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (_to, _from, saved) => saved ?? { top: 0 },
  routes: [
    { path: '/', redirect: '/restaurants' },
    { path: '/restaurants', component: () => import('@/pages/RestaurantsPage.vue') },
    { path: '/reviewers/:reviewerId', component: () => import('@/pages/ReviewerPage.vue'), props: true },
    { path: '/lists/:listId', component: () => import('@/pages/ListPage.vue'), props: true },
    { path: '/my-reviews', component: () => import('@/pages/MyReviewsPage.vue'), meta: { requiresAuth: true } },
    {
      path: '/restaurants/new',
      component: () => import('@/pages/RestaurantFormPage.vue'),
      meta: { requiresAuth: true, requiresApproval: true },
    },
    { path: '/restaurants/:restaurantId', component: () => import('@/pages/RestaurantPage.vue'), props: true },
    {
      path: '/restaurants/:restaurantId/edit',
      component: () => import('@/pages/RestaurantFormPage.vue'),
      props: true,
      meta: { requiresAuth: true, requiresApproval: true },
    },
    {
      path: '/restaurants/:restaurantId/reviews/new',
      component: () => import('@/pages/ReviewFormPage.vue'),
      props: true,
      meta: { requiresAuth: true, requiresApproval: true },
    },
    {
      path: '/restaurants/:restaurantId/reviews/:reviewId',
      component: () => import('@/pages/ReviewPage.vue'),
      props: true,
    },
    {
      path: '/restaurants/:restaurantId/reviews/:reviewId/edit',
      component: () => import('@/pages/ReviewFormPage.vue'),
      props: true,
      meta: { requiresAuth: true, requiresApproval: true },
    },
    { path: '/sign-in', component: () => import('@/pages/SignInPage.vue'), meta: { guestOnly: true } },
    { path: '/sign-up', component: () => import('@/pages/SignUpPage.vue'), meta: { guestOnly: true } },
    { path: '/confirm-email', component: () => import('@/pages/ConfirmEmailPage.vue') },
    { path: '/forgot-password', component: () => import('@/pages/ForgotPasswordPage.vue'), meta: { guestOnly: true } },
    { path: '/reset-password', component: () => import('@/pages/ResetPasswordPage.vue') },
    { path: '/account', component: () => import('@/pages/AccountPage.vue'), meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', component: ErrorPage, props: { status: 404 } },
  ],
})

router.beforeEach(async (to, from) => {
  if (to.meta.requiresAuth && !signedIn.value) {
    return signInPath(to.fullPath)
  }
  if (to.meta.requiresApproval && pendingApproval.value) {
    const firstLoad = from.matched.length === 0
    // The stored user of a first load can be from before the approval.
    if (firstLoad) {
      await refreshSession()
      if (!signedIn.value) {
        return signInPath(to.fullPath)
      }
    }
    if (tellPendingApproval()) {
      // A link on a page stays on the page. A first load has no page.
      return firstLoad ? '/restaurants' : false
    }
  }
  if (to.meta.guestOnly && signedIn.value) {
    return safeRedirect(to.query.redirect)
  }
  return true
})
