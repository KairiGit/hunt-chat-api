'use client';

import { useState } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { HeroSection } from "@/components/home/HeroSection";
import { QuickNavigation } from "@/components/home/QuickNavigation";
import { OverviewTab } from "@/components/home/tabs/OverviewTab";
import { FeaturesTab } from "@/components/home/tabs/FeaturesTab";
import { ArchitectureTab } from "@/components/home/tabs/ArchitectureTab";
import { ApiTab } from "@/components/home/tabs/ApiTab";
import { Footer } from "@/components/home/Footer";

export default function HomePage() {
  const [activeTab, setActiveTab] = useState('overview');

  return (
    <div className="space-y-8">
      <HeroSection />
      <QuickNavigation />

      {/* Documentation Tabs */}
      <section>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview">概要</TabsTrigger>
            <TabsTrigger value="features">主要機能</TabsTrigger>
            <TabsTrigger value="architecture">アーキテクチャ</TabsTrigger>
            <TabsTrigger value="api">API仕様</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4 mt-4">
            <OverviewTab />
          </TabsContent>

          <TabsContent value="features" className="space-y-4 mt-4">
            <FeaturesTab />
          </TabsContent>

          <TabsContent value="architecture" className="space-y-4 mt-4">
            <ArchitectureTab />
          </TabsContent>

          <TabsContent value="api" className="space-y-4 mt-4">
            <ApiTab />
          </TabsContent>
        </Tabs>
      </section>

      <Footer />
    </div>
  );
}
