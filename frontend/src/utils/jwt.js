import { SignJWT } from 'jose';

const JWT_SECRET = import.meta.env.VITE_JWT_SECRET || 'test-jwt-secret';

/**
 * Generate a JWT compatible with golang-jwt/jwt/v5 (HS256).
 * Claims: user_id, role, iss, iat, exp
 */
export async function generateJWT(userId) {
  const secret = new TextEncoder().encode(JWT_SECRET);
  const token = await new SignJWT({
    user_id: userId,
    role: 'driver',
  })
    .setProtectedHeader({ alg: 'HS256', typ: 'JWT' })
    .setIssuer('parkir-pintar')
    .setIssuedAt()
    .setExpirationTime('7d')
    .setSubject(userId)
    .sign(secret);
  return token;
}
