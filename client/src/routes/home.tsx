import React from "react";
import { Link } from "react-router";

export default function Home() {
  return (
    <div className="homepage">
      <header className="hero">
        <h1>Pow Hunter</h1>
        <p className="tagline">Never miss a powder day at your favorite resort</p>
        <div className="cta-buttons">
          <Link to="/resorts" className="button primary">Find Resorts</Link>
          <Link to="/signup" className="button secondary">Sign Up for Alerts</Link>
        </div>
      </header>
      
      <main>
        <section className="features">
          <h2>Why Choose Pow Hunter?</h2>
          <div className="feature-grid">
            <div className="feature">
              <h3>Save Your Favorites</h3>
              <p>Keep track of your favorite ski resorts in one place</p>
            </div>
            <div className="feature">
              <h3>Weather Forecasts</h3>
              <p>Get detailed snow and weather forecasts for your resorts</p>
            </div>
            <div className="feature">
              <h3>Text Alerts</h3>
              <p>Receive notifications when fresh powder is on the way</p>
            </div>
          </div>
        </section>
      </main>
    </div>
  );
} 